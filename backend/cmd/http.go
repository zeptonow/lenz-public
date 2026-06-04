package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"openreplay/backend/internal/config/http"
	httpServices "openreplay/backend/internal/http/services"
	"openreplay/backend/pkg/db/postgres/pool"
	"openreplay/backend/pkg/db/redis"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/metrics"
	"openreplay/backend/pkg/metrics/database"
	"openreplay/backend/pkg/metrics/web"
	"openreplay/backend/pkg/queue"
	"openreplay/backend/pkg/server"
	"openreplay/backend/pkg/server/api"
	"openreplay/backend/pkg/server/middleware"
)

func init() {
	rootCmd.AddCommand(httpCmd)
}

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Start the HTTP ingestion service",
	Run: func(cmd *cobra.Command, args []string) {
		runHTTP()
	},
}

func runHTTP() {
	ctx := context.Background()
	log := logger.New()
	cfg := http.New(log)

	webMetrics := web.New("http")
	dbMetric := database.New("http")
	metrics.New(log, append(webMetrics.List(), dbMetric.List()...))

	// Configure redisstream before creating producer
	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

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
	defer producer.Close(15000)

	pgConn, err := pool.New(dbMetric, cfg.Postgres.String())
	if err != nil {
		log.Fatal(ctx, "can't init postgres connection: %s", err)
	}
	defer pgConn.Close()

	redisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		log.Info(ctx, "no redis cache: %s", err)
	}
	defer redisClient.Close()

	services, err := httpServices.New(log, cfg, webMetrics, dbMetric, producer, pgConn, redisClient)
	if err != nil {
		log.Fatal(ctx, "failed while creating services: %s", err)
	}

	middlewares, err := middleware.NewMinimalMiddlewareBuilder(&cfg.HTTP)
	if err != nil {
		log.Fatal(ctx, "failed while creating minimal http middleware: %s", err)
	}

	router, err := api.NewRouter(log, &cfg.HTTP, api.NoPrefix, services.Handlers(), middlewares.Middlewares())
	if err != nil {
		log.Fatal(ctx, "failed while creating router: %s", err)
	}

	server.Run(ctx, log, &cfg.HTTP, router)
}
