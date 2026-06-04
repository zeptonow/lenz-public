package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	config "openreplay/backend/internal/config/images"
	"openreplay/backend/internal/images"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics"
	imagesMetrics "openreplay/backend/pkg/metrics/images"
	"openreplay/backend/pkg/objectstorage/store"
	"openreplay/backend/pkg/queue"
)

func main() {
	ctx := context.Background()
	log := logger.New()
	cfg := config.New(log)

	imageMetrics := imagesMetrics.New("images")
	metrics.New(log, imageMetrics.List())

	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

	objStore, err := store.NewStore(&cfg.ObjectsConfig)
	if err != nil {
		log.Fatal(ctx, "can't init object storage: %s", err)
	}

	srv, err := images.New(cfg, log, objStore, imageMetrics)
	if err != nil {
		log.Fatal(ctx, "can't init images service: %s", err)
	}

	workDir := cfg.FSDir

	imagesIter := messages.NewImagesMessageIterator(func(data []byte, sessID uint64) {
		checkSessionEnd := func(data []byte) (messages.Message, error) {
			reader := messages.NewBytesReader(data)
			msgType, err := reader.ReadUint()
			if err != nil {
				return nil, err
			}
			if msgType != messages.MsgMobileSessionEnd {
				return nil, fmt.Errorf("not a mobile session end message")
			}
			msg, err := messages.ReadMessage(msgType, reader)
			if err != nil {
				return nil, fmt.Errorf("read message err: %s", err)
			}
			return msg, nil
		}
		sessCtx := context.WithValue(context.Background(), "sessionID", fmt.Sprintf("%d", sessID))

		if _, err := checkSessionEnd(data); err == nil {
			if err := srv.PackScreenshots(sessCtx, sessID, workDir+"/screenshots/"+strconv.FormatUint(sessID, 10)+"/"); err != nil {
				log.Error(sessCtx, "can't pack screenshots: %s", err)
			}
		} else {
			if err := srv.Process(sessCtx, sessID, data); err != nil {
				log.Error(sessCtx, "can't process screenshots: %s", err)
			}
		}
	}, nil, true)
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
	consumer, err := queue.NewRawTopicConsumer(log, cfg.GroupImageStorage, cfg.TopicRawImages, imagesIter, nil, kafkaOpts)
	if err != nil {
		log.Error(ctx, "can't init message consumer: %s", err)
	}

	log.Info(ctx, "Images service started")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	counterTick := time.Tick(time.Second * 30)
	for {
		select {
		case sig := <-sigchan:
			log.Info(ctx, "Caught signal %v: terminating", sig)
			srv.Wait()
			if consumer != nil {
				consumer.Close()
			}
			os.Exit(0)
		case <-counterTick:
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
				log.Error(ctx, "Error on images consumption: %v", err)
			}
		}
	}
}
