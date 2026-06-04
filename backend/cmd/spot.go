package cmd

import (
	"context"

	"github.com/spf13/cobra"

	spotConfig "openreplay/backend/internal/config/spot"
	"openreplay/backend/pkg/db/postgres/pool"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/metrics"
	databaseMetrics "openreplay/backend/pkg/metrics/database"
	spotMetrics "openreplay/backend/pkg/metrics/spot"
	"openreplay/backend/pkg/metrics/web"
	"openreplay/backend/pkg/server"
	"openreplay/backend/pkg/server/api"
	"openreplay/backend/pkg/server/middleware"
	"openreplay/backend/pkg/spot"
	"openreplay/backend/pkg/spot/keys"
)

func init() {
	rootCmd.AddCommand(spotCmd)
}

var spotCmd = &cobra.Command{
	Use:   "spot",
	Short: "Start the Spot recording service",
	Run: func(cmd *cobra.Command, args []string) {
		runSpot()
	},
}

func runSpot() {
	ctx := context.Background()
	log := logger.New()
	cfg := spotConfig.New(log)

	webMetric := web.New("spot")
	spotMetric := spotMetrics.New("spot")
	dbMetric := databaseMetrics.New("spot")
	metrics.New(log, append(webMetric.List(), append(spotMetric.List(), dbMetric.List()...)...))

	pgPool, err := pool.New(dbMetric, cfg.Postgres.String())
	if err != nil {
		log.Fatal(ctx, "can't init postgres connection: %s", err)
	}
	defer pgPool.Close()

	spotKeys := keys.NewKeys(log, pgPool)
	services, err := spot.NewServiceBuilder(log, cfg, webMetric, spotMetric, pgPool, spotKeys)
	if err != nil {
		log.Fatal(ctx, "can't init services: %s", err)
	}

	prefix := api.NoPrefix

	middlewares, err := middleware.NewMiddlewareBuilder(log, cfg.JWTSecret, cfg.GoInterserviceAPIKey, &cfg.HTTP, &cfg.RateLimiter, prefix, pgPool, dbMetric, services.Handlers(), nil, nil, &cfg.JWTSpotSecret, spotKeys)
	if err != nil {
		log.Fatal(ctx, "can't init middlewares: %s", err)
	}

	router, err := api.NewRouter(log, &cfg.HTTP, prefix, services.Handlers(), middlewares.Middlewares())
	if err != nil {
		log.Fatal(ctx, "failed while creating router: %s", err)
	}

	server.Run(ctx, log, &cfg.HTTP, router)
}
