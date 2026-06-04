package cmd

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	config "openreplay/backend/internal/config/sink"
	"openreplay/backend/internal/sink/assetscache"
	"openreplay/backend/internal/sink/sessionwriter"
	"openreplay/backend/internal/storage"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics"
	"openreplay/backend/pkg/metrics/sink"
	"openreplay/backend/pkg/queue"
	"openreplay/backend/pkg/queue/types"
	"openreplay/backend/pkg/url/assets"
)

func init() {
	rootCmd.AddCommand(sinkCmd)
}

var sinkCmd = &cobra.Command{
	Use:   "sink",
	Short: "Start the Sink consumer service",
	Run: func(cmd *cobra.Command, args []string) {
		runSink()
	},
}

func runSink() {
	ctx := context.Background()
	log := logger.New()
	cfg := config.New(log)

	sinkMetrics := sink.New("sink")
	metrics.New(log, sinkMetrics.List())

	if _, err := os.Stat(cfg.FsDir); os.IsNotExist(err) {
		log.Fatal(ctx, "%v doesn't exist. %v", cfg.FsDir, err)
	}

	// Configure redisstream before creating producer/consumer
	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

	writer := sessionwriter.NewWriter(log, cfg.FsUlimit, cfg.FsDir, cfg.FileBuffer, cfg.SyncTimeout)

	var rawKafka *queue.RawKafkaOpts
	if cfg.KafkaEnabled() {
		var kerr error
		rawKafka, kerr = queue.NewRawKafkaOpts(queue.KafkaSettingsFromProducerConsumer(
			cfg.ProducerBrokers(), cfg.ProducerSaslUsername(), cfg.ProducerSaslPassword(),
			cfg.ConsumerBrokers(), cfg.ConsumerSaslUsername(), cfg.ConsumerSaslPassword(),
			cfg.KafkaConsumerPartition,
		), func(o *queue.RawKafkaOpts) {
			o.TopicRawWeb = cfg.TopicRawWeb
			o.TopicRawMobile = cfg.TopicRawMobile
			o.TopicTrigger = cfg.TopicTrigger
			o.TopicMobileTrigger = cfg.TopicMobileTrigger
		})
		if kerr != nil {
			log.Error(ctx, "kafka configuration: %v", kerr)
			rawKafka = nil
		}
	}
	producer := queue.NewProducer(cfg.MessageSizeLimit, true, rawKafka)
	defer producer.Close(cfg.ProducerCloseTimeout)
	rewriter, err := assets.NewRewriter(cfg.AssetsOrigin)
	if err != nil {
		log.Fatal(ctx, "can't init rewriter: %s", err)
	}
	assetMessageHandler := assetscache.New(log, cfg, rewriter, producer, sinkMetrics)
	counter := storage.NewLogCounter()

	var (
		sessionID    uint64
		messageIndex = make([]byte, 8)
		domBuffer    = bytes.NewBuffer(make([]byte, 1024))
		devBuffer    = bytes.NewBuffer(make([]byte, 1024))
	)

	domBuffer.Reset()
	devBuffer.Reset()

	msgHandler := func(msg messages.Message) {
		if msg == nil {
			if domBuffer.Len() <= 0 && devBuffer.Len() <= 0 {
				return
			}
			sinkMetrics.RecordWrittenBytes(float64(domBuffer.Len()), "dom")
			sinkMetrics.RecordWrittenBytes(float64(devBuffer.Len()), "devtools")

			if err := writer.Write(sessionID, domBuffer.Bytes(), devBuffer.Bytes()); err != nil {
				sessCtx := context.WithValue(context.Background(), "sessionID", sessionID)
				log.Error(sessCtx, "writer error: %s", err)
			}

			domBuffer.Reset()
			devBuffer.Reset()
			sessionID = 0
			return
		}

		sinkMetrics.IncreaseTotalMessages()
		sessCtx := context.WithValue(context.Background(), "sessionID", msg.SessionID())

		if msg.TypeID() == messages.MsgSessionEnd || msg.TypeID() == messages.MsgMobileSessionEnd {
			if err := producer.Produce(cfg.TopicTrigger, msg.SessionID(), msg.Encode()); err != nil {
				log.Error(sessCtx, "can't send SessionEnd to trigger topic: %s", err)
			}
			if msg.TypeID() == messages.MsgMobileSessionEnd {
				if err := producer.Produce(cfg.TopicMobileTrigger, msg.SessionID(), msg.Encode()); err != nil {
					log.Error(sessCtx, "can't send MobileSessionEnd to mobile trigger topic: %s", err)
				}
			}
			writer.Close(msg.SessionID())
			return
		}

		if msg.TypeID() == messages.MsgSetNodeAttributeURLBased ||
			msg.TypeID() == messages.MsgSetCSSDataURLBased ||
			msg.TypeID() == messages.MsgAdoptedSSReplaceURLBased ||
			msg.TypeID() == messages.MsgAdoptedSSInsertRuleURLBased {
			m := msg.Decode()
			if m == nil {
				log.Error(sessCtx, "assets decode err, info: %s", msg.Meta().Batch().Info())
				return
			}
			msg = assetMessageHandler.ParseAssets(m)
		}

		if !messages.IsReplayerType(msg.TypeID()) {
			return
		}

		ts := msg.Meta().Timestamp
		if ts == 0 {
			log.Warn(sessCtx, "zero ts in msgType: %d", msg.TypeID())
		} else {
			counter.Update(msg.SessionID(), time.UnixMilli(int64(ts)))
		}

		data := msg.Encode()
		if data == nil {
			return
		}

		if sessionID == 0 {
			sessionID = msg.SessionID()
		}

		binary.LittleEndian.PutUint64(messageIndex, msg.Meta().Index)

		if messages.IsDOMType(msg.TypeID()) {
			domBuffer.Write(messageIndex)
			domBuffer.Write(msg.Encode())
		}

		if !messages.IsDOMType(msg.TypeID()) || msg.TypeID() == messages.MsgTimestamp || msg.TypeID() == messages.MsgTabData {
			devBuffer.Write(messageIndex)
			devBuffer.Write(msg.Encode())
		}

		sinkMetrics.IncreaseWrittenMessages()
		sinkMetrics.RecordMessageSize(float64(len(msg.Encode())))
	}

	sinkIterator := messages.NewSinkMessageIterator(log, msgHandler, nil, false, sinkMetrics)
	rebalance := func(t types.RebalanceType, partitions []uint64) {
		s := time.Now()
		writer.Sync()
		log.Info(ctx, "manual sync finished, dur: %d", time.Now().Sub(s).Milliseconds())
	}

	if rawKafka != nil && rawKafka.ConsumerEnabled() {
		if err := queue.SetConsumerPartitions(rawKafka); err != nil {
			log.Error(ctx, "kafka consumer partitions: %v", err)
			rawKafka = nil
		}
	}

	consumerWeb, err := queue.NewRawTopicConsumer(
		log,
		cfg.GroupSink,
		cfg.TopicRawWeb,
		sinkIterator,
		rebalance,
		rawKafka,
	)
	if err != nil {
		log.Error(ctx, "can't init raw web consumer: %s", err)
	}

	consumerMobile, err := queue.NewRawTopicConsumer(
		log,
		cfg.GroupSinkMobile,
		cfg.TopicRawMobile,
		sinkIterator,
		rebalance,
		rawKafka,
	)
	if err != nil {
		if consumerWeb != nil {
			consumerWeb.Close()
			consumerWeb = nil
		}
		log.Error(ctx, "can't init raw mobile consumer: %s", err)
	}

	log.Info(ctx, "sink service started")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	tick := time.Tick(10 * time.Second)
	tickInfo := time.Tick(30 * time.Second)
	for {
		select {
		case sig := <-sigchan:
			log.Info(ctx, "Caught signal %v: terminating", sig)
			writer.Stop()
			if consumerWeb != nil {
				if err := consumerWeb.Commit(); err != nil {
					log.Error(ctx, "can't commit raw web: %s", err)
				}
				consumerWeb.Close()
			}
			if consumerMobile != nil {
				if err := consumerMobile.Commit(); err != nil {
					log.Error(ctx, "can't commit raw mobile: %s", err)
				}
				consumerMobile.Close()
			}
			os.Exit(0)
		case <-tick:
			if consumerWeb != nil {
				if err := consumerWeb.Commit(); err != nil {
					log.Error(ctx, "can't commit raw web: %s", err)
				}
			}
			if consumerMobile != nil {
				if err := consumerMobile.Commit(); err != nil {
					log.Error(ctx, "can't commit raw mobile: %s", err)
				}
			}
		case <-tickInfo:
			log.Info(ctx, "%s", counter.Log())
			log.Info(ctx, "writer: %s", writer.Info())
		default:
			if consumerWeb != nil {
				if err := consumerWeb.ConsumeNext(); err != nil {
					log.Error(ctx, "error on raw web consumption: %v", err)
				}
			}
			if consumerMobile != nil {
				if err := consumerMobile.ConsumeNext(); err != nil {
					log.Error(ctx, "error on raw mobile consumption: %v", err)
				}
			}
			if consumerWeb == nil && consumerMobile == nil {
				time.Sleep(time.Second)
			}
		}
	}
}
