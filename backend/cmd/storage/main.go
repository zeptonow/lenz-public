package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "openreplay/backend/internal/config/storage"
	"openreplay/backend/internal/storage"
	"openreplay/backend/pkg/failover"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics"
	storageMetrics "openreplay/backend/pkg/metrics/storage"
	"openreplay/backend/pkg/objectstorage/store"
	"openreplay/backend/pkg/queue"
)

func main() {
	ctx := context.Background()
	log := logger.New()
	cfg := config.New(log)

	storageMetric := storageMetrics.New("storage")
	metrics.New(log, storageMetric.List())

	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

	objStore, err := store.NewStore(&cfg.ObjectsConfig)
	if err != nil {
		log.Fatal(ctx, "can't init object storage: %s", err)
	}
	srv, err := storage.New(cfg, log, objStore, storageMetric)
	if err != nil {
		log.Fatal(ctx, "can't init storage service: %s", err)
	}

	counter := storage.NewLogCounter()
	sessionFinder, err := failover.NewSessionFinder(log, cfg, srv)
	if err != nil {
		log.Fatal(ctx, "can't init sessionFinder module: %s", err)
	}

	triggerIter := messages.NewMessageIterator(
		log,
		func(msg messages.Message) {
			// Convert MobileSessionEnd to SessionEnd
			if msg.TypeID() == messages.MsgMobileSessionEnd {
				mobileEnd, oldMeta := msg.(*messages.MobileSessionEnd), msg.Meta()
				msg = &messages.SessionEnd{
					Timestamp: mobileEnd.Timestamp,
				}
				msg.Meta().SetMeta(oldMeta)
			}
			sessCtx := context.WithValue(context.Background(), "sessionID", fmt.Sprintf("%d", msg.SessionID()))
			// Process session to save mob files to s3
			sesEnd := msg.(*messages.SessionEnd)
			if err := srv.Process(sessCtx, sesEnd); err != nil {
				log.Error(sessCtx, "process session err: %s", err)
				sessionFinder.Find(msg.SessionID(), sesEnd.Timestamp)
			}
			// Log timestamp of last processed session
			counter.Update(msg.SessionID(), time.UnixMilli(msg.Meta().Batch().Timestamp()))
		},
		[]int{messages.MsgSessionEnd, messages.MsgMobileSessionEnd},
		true,
	)
	var kafkaOpts *queue.RawKafkaOpts
	if cfg.KafkaEnabled() {
		var kerr error
		kafkaOpts, kerr = queue.NewRawKafkaOpts(queue.KafkaSettingsFromProducerConsumer(
			cfg.ProducerBrokers(), cfg.ProducerSaslUsername(), cfg.ProducerSaslPassword(),
			cfg.ConsumerBrokers(), cfg.ConsumerSaslUsername(), cfg.ConsumerSaslPassword(),
			cfg.KafkaConsumerPartition,
		), nil)
		if kerr != nil {
			log.Error(ctx, "kafka configuration: %v", kerr)
			kafkaOpts = nil
		} else if err := queue.SetConsumerPartitions(kafkaOpts); err != nil {
			log.Error(ctx, "kafka consumer partitions: %v", err)
			kafkaOpts = nil
		}
	}
	consumer, err := queue.NewRawTopicConsumer(log, cfg.GroupStorage, cfg.TopicTrigger, triggerIter, nil, kafkaOpts)
	if err != nil {
		log.Error(ctx, "can't init message consumer: %s", err)
	}

	log.Info(ctx, "Storage service started")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	counterTick := time.Tick(time.Second * 30)
	for {
		select {
		case sig := <-sigchan:
			log.Info(ctx, "caught signal %v: terminating", sig)
			sessionFinder.Stop()
			srv.Wait()
			if consumer != nil {
				consumer.Close()
			}
			os.Exit(0)
		case <-counterTick:
			go log.Info(ctx, "%s", counter.Log())
			srv.Wait()
			if consumer != nil {
				if err := consumer.Commit(); err != nil {
					log.Error(ctx, "can't commit messages: %s", err)
				}
			}
		default:
			if consumer == nil {
				time.Sleep(time.Second)
				continue
			}
			if err := consumer.ConsumeNext(); err != nil {
				log.Error(ctx, "error on consumption: %v", err)
			}
		}
	}
}
