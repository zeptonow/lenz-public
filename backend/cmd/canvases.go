package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"openreplay/backend/internal/canvases"
	config "openreplay/backend/internal/config/canvases"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics"
	canvasesMetrics "openreplay/backend/pkg/metrics/canvas"
	"openreplay/backend/pkg/objectstorage/store"
	"openreplay/backend/pkg/queue"
	"openreplay/backend/pkg/queue/types"
)

func init() {
	rootCmd.AddCommand(canvasesCmd)
}

var canvasesCmd = &cobra.Command{
	Use:   "canvases",
	Short: "Start the Canvases service",
	Run: func(cmd *cobra.Command, args []string) {
		runCanvases()
	},
}

func runCanvases() {
	ctx := context.Background()
	log := logger.New()
	cfg := config.New(log)

	canvasMetrics := canvasesMetrics.New("canvases")
	metrics.New(log, canvasMetrics.List())

	// Configure redisstream before creating producer/consumer
	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

	objStore, err := store.NewStore(&cfg.ObjectsConfig)
	if err != nil {
		log.Fatal(ctx, "can't init object storage: %s", err)
	}

	var kafkaOpts *queue.RawKafkaOpts
	if cfg.KafkaEnabled() {
		var kerr error
		kafkaOpts, kerr = queue.NewRawKafkaOpts(queue.KafkaSettingsFromProducerConsumer(
			cfg.ProducerBrokers(), cfg.ProducerSaslUsername(), cfg.ProducerSaslPassword(),
			cfg.ConsumerBrokers(), cfg.ConsumerSaslUsername(), cfg.ConsumerSaslPassword(),
			cfg.KafkaConsumerPartition,
		), func(o *queue.RawKafkaOpts) {
			o.TopicCanvasImages = cfg.TopicCanvasImages
			o.TopicCanvasTrigger = cfg.TopicCanvasTrigger
		})
		if kerr != nil {
			log.Error(ctx, "kafka configuration: %v", kerr)
			kafkaOpts = nil
		}
	}
	producer := queue.NewProducer(cfg.MessageSizeLimit, true, kafkaOpts)
	defer producer.Close(15000)

	srv, err := canvases.New(cfg, log, objStore, producer, canvasMetrics)
	if err != nil {
		log.Fatal(ctx, "can't init canvases service: %s", err)
	}

	// Create canvas directory under FSDir (CanvasDir is a relative segment, e.g. "canvas", not CWD-relative).
	if cfg.CanvasDir != "" {
		canvasPath := filepath.Join(cfg.FSDir, cfg.CanvasDir)
		if _, err := os.Stat(canvasPath); os.IsNotExist(err) {
			if err := os.MkdirAll(canvasPath, 0755); err != nil {
				log.Fatal(ctx, "can't create canvas directory %s: %v", canvasPath, err)
			}
		}
	}

	canvasIter := messages.NewImagesMessageIterator(func(data []byte, sessID uint64) {
		isSessionEnd := func(data []byte) bool {
			reader := messages.NewBytesReader(data)
			msgType, err := reader.ReadUint()
			if err != nil {
				return false
			}
			if msgType != messages.MsgSessionEnd {
				return false
			}
			_, err = messages.ReadMessage(msgType, reader)
			return err == nil
		}
		isTriggerEvent := func(data []byte) (string, string, bool) {
			reader := messages.NewBytesReader(data)
			msgType, err := reader.ReadUint()
			if err != nil {
				return "", "", false
			}
			if msgType != messages.MsgCustomEvent {
				return "", "", false
			}
			msg, err := messages.ReadMessage(msgType, reader)
			if err != nil {
				return "", "", false
			}
			customEvent := msg.(*messages.CustomEvent)
			return customEvent.Payload, customEvent.Name, true
		}
		sessCtx := context.WithValue(context.Background(), "sessionID", sessID)

		if isSessionEnd(data) {
			if err := srv.PrepareSessionCanvases(sessCtx, sessID); err != nil {
				if !strings.Contains(err.Error(), "no such file or directory") {
					log.Error(sessCtx, "can't pack session's canvases: %s", err)
				}
			}
		} else if path, name, ok := isTriggerEvent(data); ok {
			if err := srv.ProcessSessionCanvas(sessCtx, sessID, path, name); err != nil {
				log.Error(sessCtx, "can't process session's canvas: %s", err)
			}
		} else {
			if err := srv.SaveCanvasToDisk(sessCtx, sessID, data); err != nil {
				log.Error(sessCtx, "can't process canvas image: %s", err)
			}
		}
	}, nil, true)

	var canvasConsumerRedis types.Consumer
	var consumerCanvasImages, consumerCanvasTrigger types.Consumer
	var consErr error
	if kafkaOpts == nil || !kafkaOpts.ConsumerEnabled() {
		canvasConsumerRedis, consErr = queue.NewConsumer(
			log,
			cfg.GroupCanvasImage,
			[]string{cfg.TopicCanvasImages, cfg.TopicCanvasTrigger},
			canvasIter,
			false,
			cfg.MessageSizeLimit,
			nil,
			types.NoReadBackGap,
		)
	} else {
		if err := queue.SetConsumerPartitions(kafkaOpts); err != nil {
			log.Error(ctx, "kafka consumer partitions: %v", err)
			kafkaOpts = nil
			canvasConsumerRedis, consErr = queue.NewConsumer(
				log,
				cfg.GroupCanvasImage,
				[]string{cfg.TopicCanvasImages, cfg.TopicCanvasTrigger},
				canvasIter,
				false,
				cfg.MessageSizeLimit,
				nil,
				types.NoReadBackGap,
			)
		} else {
			locked := messages.NewLockedIterator(canvasIter)
			var errImg, errTrg error
			consumerCanvasImages, errImg = queue.NewRawTopicConsumer(log, cfg.GroupCanvasImage, cfg.TopicCanvasImages, locked, nil, kafkaOpts)
			if errImg != nil {
				log.Error(ctx, "can't init canvas images consumer: %s", errImg)
			}
			consumerCanvasTrigger, errTrg = queue.NewRawTopicConsumer(log, cfg.GroupCanvasTrigger, cfg.TopicCanvasTrigger, locked, nil, kafkaOpts)
			if errTrg != nil {
				if consumerCanvasImages != nil {
					consumerCanvasImages.Close()
					consumerCanvasImages = nil
				}
				log.Error(ctx, "can't init canvas trigger consumer: %s", errTrg)
			}
		}
	}
	if consErr != nil {
		log.Error(ctx, "can't init canvases service: %s", consErr)
	}

	log.Info(ctx, "canvases service started")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	counterTick := time.Tick(time.Second * 30)
	for {
		select {
		case sig := <-sigchan:
			log.Info(ctx, "caught signal %v: terminating", sig)
			srv.Wait()
			if kafkaOpts == nil || !kafkaOpts.ConsumerEnabled() {
				if canvasConsumerRedis != nil {
					canvasConsumerRedis.Close()
				}
			} else {
				if consumerCanvasImages != nil {
					if err := consumerCanvasImages.Commit(); err != nil {
						log.Error(ctx, "can't commit canvas images: %s", err)
					}
					consumerCanvasImages.Close()
				}
				if consumerCanvasTrigger != nil {
					if err := consumerCanvasTrigger.Commit(); err != nil {
						log.Error(ctx, "can't commit canvas trigger: %s", err)
					}
					consumerCanvasTrigger.Close()
				}
			}
			os.Exit(0)
		case <-counterTick:
			srv.Wait()
			if kafkaOpts == nil || !kafkaOpts.ConsumerEnabled() {
				if canvasConsumerRedis != nil {
					if err := canvasConsumerRedis.Commit(); err != nil {
						log.Error(ctx, "can't commit messages: %s", err)
					}
				}
			} else {
				if consumerCanvasImages != nil {
					if err := consumerCanvasImages.Commit(); err != nil {
						log.Error(ctx, "can't commit canvas images: %s", err)
					}
				}
				if consumerCanvasTrigger != nil {
					if err := consumerCanvasTrigger.Commit(); err != nil {
						log.Error(ctx, "can't commit canvas trigger: %s", err)
					}
				}
			}
		default:
			if kafkaOpts == nil || !kafkaOpts.ConsumerEnabled() {
				if canvasConsumerRedis != nil {
					if err := canvasConsumerRedis.ConsumeNext(); err != nil {
						log.Error(ctx, "can't consume next message: %s", err)
					}
				}
			} else {
				if consumerCanvasImages != nil {
					if err := consumerCanvasImages.ConsumeNext(); err != nil {
						log.Error(ctx, "error on canvas images consumption: %v", err)
					}
				}
				if consumerCanvasTrigger != nil {
					if err := consumerCanvasTrigger.ConsumeNext(); err != nil {
						log.Error(ctx, "error on canvas trigger consumption: %v", err)
					}
				}
				if consumerCanvasImages == nil && consumerCanvasTrigger == nil {
					time.Sleep(time.Second)
				}
			}
		}
	}
}
