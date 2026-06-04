package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	config "openreplay/backend/internal/config/ender"
	"openreplay/backend/internal/ender"
	"openreplay/backend/internal/storage"
	"openreplay/backend/pkg/db/postgres/pool"
	"openreplay/backend/pkg/db/redis"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/memory"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics"
	"openreplay/backend/pkg/metrics/database"
	enderMetrics "openreplay/backend/pkg/metrics/ender"
	"openreplay/backend/pkg/projects"
	"openreplay/backend/pkg/queue"
	"openreplay/backend/pkg/queue/types"
	"openreplay/backend/pkg/sessions"
)

func init() {
	rootCmd.AddCommand(enderCmd)
}

var enderCmd = &cobra.Command{
	Use:   "ender",
	Short: "Start the Ender session-end service",
	Run: func(cmd *cobra.Command, args []string) {
		runEnder()
	},
}

type enderSessionEndType int

const (
	enderFailedSessionEnd enderSessionEndType = iota + 1
	enderDuplicatedSessionEnd
	enderNegativeDuration
	enderShorterDuration
	enderNewSessionEnd
	enderNoSessionInDB
)

func runEnder() {
	ctx := context.Background()
	log := logger.New()
	cfg := config.New(log)

	dbMetric := database.New("ender")
	enderMetric := enderMetrics.New("ender")
	metrics.New(log, append(enderMetric.List(), dbMetric.List()...))

	// Configure redisstream before creating producer/consumer
	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

	pgConn, err := pool.New(dbMetric, cfg.Postgres.String())
	if err != nil {
		log.Fatal(ctx, "can't init postgres connection: %s", err)
	}
	defer pgConn.Close()

	redisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		log.Warn(ctx, "can't init redis connection: %s", err)
	}
	defer redisClient.Close()

	projManager := projects.New(log, pgConn, redisClient, dbMetric)
	sessManager := sessions.New(log, pgConn, projManager, redisClient, dbMetric)

	sessionEndGenerator, err := ender.New(enderMetric, ender.EVENTS_SESSION_END_TIMEOUT, cfg.PartitionsNumber)
	if err != nil {
		log.Fatal(ctx, "can't init ender service: %s", err)
	}

	mobileMessages := []int{90, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 107, 110, 111}

	var rawK *queue.RawKafkaOpts
	if cfg.KafkaEnabled() {
		var kerr error
		rawK, kerr = queue.NewRawKafkaOpts(queue.KafkaSettingsFromProducerConsumer(
			cfg.ProducerBrokers(), cfg.ProducerSaslUsername(), cfg.ProducerSaslPassword(),
			cfg.ConsumerBrokers(), cfg.ConsumerSaslUsername(), cfg.ConsumerSaslPassword(),
			cfg.KafkaConsumerPartition,
		), func(o *queue.RawKafkaOpts) {
			o.TopicRawWeb = cfg.TopicRawWeb
			o.TopicRawMobile = cfg.TopicRawMobile
			o.TopicRawImages = cfg.TopicRawImages
			o.TopicCanvasImages = cfg.TopicCanvasImages
		})
		if kerr != nil {
			log.Error(ctx, "kafka configuration: %v", kerr)
			rawK = nil
		}
	}
	producer := queue.NewProducer(cfg.MessageSizeLimit, true, rawK)
	if rawK != nil && rawK.ConsumerEnabled() {
		if err := queue.SetConsumerPartitions(rawK); err != nil {
			log.Error(ctx, "kafka consumer partitions: %v", err)
			rawK = nil
		}
	}

	enderIter := messages.NewEnderMessageIterator(
		log,
		func(msg messages.Message) { sessionEndGenerator.UpdateSession(msg) },
		append([]int{messages.MsgTimestamp}, mobileMessages...),
		false,
	)

	var consumerRedis types.Consumer
	var consumerRawWeb, consumerRawMobile types.Consumer
	var consErr error
	if rawK == nil || !rawK.ConsumerEnabled() {
		consumerRedis, consErr = queue.NewConsumer(
			log,
			cfg.GroupEnder,
			[]string{cfg.TopicRawWeb, cfg.TopicRawMobile},
			enderIter,
			false,
			cfg.MessageSizeLimit,
			func(t types.RebalanceType, partitions []uint64) {
				if t == types.RebalanceTypeRevoke {
					sessionEndGenerator.Disable()
				} else {
					sessionEndGenerator.ActivePartitions(partitions)
					sessionEndGenerator.Enable()
				}
			},
			-ender.EVENTS_BACK_COMMIT_GAP,
		)
	} else {
		locked := messages.NewLockedIterator(enderIter)
		var errW, errM error
		consumerRawWeb, errW = queue.NewRawTopicConsumer(log, cfg.GroupEnder, cfg.TopicRawWeb, locked, nil, rawK)
		if errW != nil {
			log.Error(ctx, "can't init ender raw web consumer: %s", errW)
		}
		consumerRawMobile, errM = queue.NewRawTopicConsumer(log, cfg.GroupEnderMobile, cfg.TopicRawMobile, locked, nil, rawK)
		if errM != nil {
			if consumerRawWeb != nil {
				consumerRawWeb.Close()
				consumerRawWeb = nil
			}
			log.Error(ctx, "can't init ender raw mobile consumer: %s", errM)
		}
	}
	if consErr != nil {
		log.Error(ctx, "can't init message consumer: %s", consErr)
	}

	memoryManager, err := memory.NewManager(log, cfg.MemoryLimitMB, cfg.MaxMemoryUsage)
	if err != nil {
		log.Fatal(ctx, "can't init memory manager: %s", err)
	}

	log.Info(ctx, "Ender service started")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	tick := time.Tick(ender.EVENTS_COMMIT_INTERVAL * time.Millisecond)
	for {
		select {
		case sig := <-sigchan:
			log.Info(ctx, "Caught signal %v: terminating", sig)
			producer.Close(cfg.ProducerTimeout)
			if rawK == nil || !rawK.ConsumerEnabled() {
				if err := consumerRedis.CommitBack(ender.EVENTS_BACK_COMMIT_GAP); err != nil {
					log.Error(ctx, "can't commit messages with offset: %s", err)
				}
				consumerRedis.Close()
			} else {
				if consumerRawWeb != nil {
					if err := consumerRawWeb.CommitBack(ender.EVENTS_BACK_COMMIT_GAP); err != nil {
						log.Error(ctx, "can't commit ender raw web with offset: %s", err)
					}
					consumerRawWeb.Close()
				}
				if consumerRawMobile != nil {
					if err := consumerRawMobile.CommitBack(ender.EVENTS_BACK_COMMIT_GAP); err != nil {
						log.Error(ctx, "can't commit ender raw mobile with offset: %s", err)
					}
					consumerRawMobile.Close()
				}
			}
			os.Exit(0)
		case <-tick:
			sessionEndGenerator.HandleEndedSessions(func(sessionID uint64, timestamp uint64) (bool, int) {
				sessCtx := context.WithValue(context.Background(), "sessionID", fmt.Sprintf("%d", sessionID))
				msg := &messages.SessionEnd{Timestamp: timestamp}
				currDuration, err := sessManager.GetDuration(sessionID)
				if err != nil {
					log.Error(sessCtx, "getSessionDuration failed, err: %s", err)
				}
				sess, err := sessManager.Get(sessionID)
				if err != nil {
					log.Error(sessCtx, "can't get session from database to compare durations, err: %s", err)
				} else {
					newDur := timestamp - sess.Timestamp
					if currDuration == newDur {
						return true, int(enderDuplicatedSessionEnd)
					}
					if currDuration > newDur {
						return true, int(enderShorterDuration)
					}
				}
				_, err = sessManager.UpdateDuration(sessionID, msg.Timestamp)
				if err != nil {
					if strings.Contains(err.Error(), "integer out of range") {
						return true, int(enderFailedSessionEnd)
					}
					if strings.Contains(err.Error(), "is less than zero for uint64") {
						return true, int(enderNegativeDuration)
					}
					if strings.Contains(err.Error(), "no rows in result set") {
						return true, int(enderNoSessionInDB)
					}
					log.Error(sessCtx, "can't update session duration, err: %s", err)
					return false, 0
				}
				if cfg.UseEncryption {
					if key := storage.GenerateEncryptionKey(); key != nil {
						if err := sessManager.UpdateEncryptionKey(sessionID, key); err != nil {
							log.Warn(sessCtx, "can't save session encryption key: %s", err)
						} else {
							msg.EncryptionKey = string(key)
						}
					}
				}
				if sess != nil && (sess.Platform == "ios" || sess.Platform == "android") {
					mobileMsg := &messages.MobileSessionEnd{Timestamp: timestamp}
					if err := producer.Produce(cfg.TopicRawMobile, sessionID, mobileMsg.Encode()); err != nil {
						log.Error(sessCtx, "can't send MobileSessionEnd: %s", err)
						return false, 0
					}
					if err := producer.Produce(cfg.TopicRawImages, sessionID, mobileMsg.Encode()); err != nil {
						log.Error(sessCtx, "can't send MobileSessionEnd to canvas: %s", err)
					}
				} else {
					if err := producer.Produce(cfg.TopicRawWeb, sessionID, msg.Encode()); err != nil {
						log.Error(sessCtx, "can't send sessionEnd to raw topic: %s", err)
						return false, 0
					}
					if err := producer.Produce(cfg.TopicCanvasImages, sessionID, msg.Encode()); err != nil {
						log.Error(sessCtx, "can't send sessionEnd to canvas: %s", err)
					}
				}
				return true, int(enderNewSessionEnd)
			})
			producer.Flush(cfg.ProducerTimeout)
			if rawK == nil || !rawK.ConsumerEnabled() {
				if err := consumerRedis.CommitBack(ender.EVENTS_BACK_COMMIT_GAP); err != nil {
					log.Error(ctx, "can't commit messages with offset: %s", err)
				}
			} else {
				if consumerRawWeb != nil {
					if err := consumerRawWeb.CommitBack(ender.EVENTS_BACK_COMMIT_GAP); err != nil {
						log.Error(ctx, "can't commit ender raw web with offset: %s", err)
					}
				}
				if consumerRawMobile != nil {
					if err := consumerRawMobile.CommitBack(ender.EVENTS_BACK_COMMIT_GAP); err != nil {
						log.Error(ctx, "can't commit ender raw mobile with offset: %s", err)
					}
				}
			}
		default:
			if !memoryManager.HasFreeMemory() {
				continue
			}
			if rawK == nil || !rawK.ConsumerEnabled() {
				if consumerRedis != nil {
					if err := consumerRedis.ConsumeNext(); err != nil {
						log.Error(ctx, "error on consuming: %s", err)
					}
				}
			} else {
				if consumerRawWeb != nil {
					if err := consumerRawWeb.ConsumeNext(); err != nil {
						log.Error(ctx, "error on ender raw web consumption: %v", err)
					}
				}
				if consumerRawMobile != nil {
					if err := consumerRawMobile.ConsumeNext(); err != nil {
						log.Error(ctx, "error on ender raw mobile consumption: %v", err)
					}
				}
				if consumerRawWeb == nil && consumerRawMobile == nil {
					time.Sleep(time.Second)
				}
			}
		}
	}
}
