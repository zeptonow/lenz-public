package cmd

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/spf13/cobra"

	sessionConfig "openreplay/backend/internal/config/api"
	apiService "openreplay/backend/pkg/api"
	"openreplay/backend/pkg/canvas"
	"openreplay/backend/pkg/db/clickhouse"
	"openreplay/backend/pkg/db/postgres/pool"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/metrics"
	"openreplay/backend/pkg/metrics/database"
	"openreplay/backend/pkg/metrics/web"
	"openreplay/backend/pkg/objectstorage/store"
	"openreplay/backend/pkg/projects"
	"openreplay/backend/pkg/server"
	"openreplay/backend/pkg/server/api"
	"openreplay/backend/pkg/server/middleware"
	"openreplay/backend/pkg/server/tenant"
)

func init() {
	rootCmd.AddCommand(apiCmd)
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Start the Go API service",
	Run: func(cmd *cobra.Command, args []string) {
		runAPI()
	},
}

func runAPI() {
	ctx := context.Background()
	log := logger.New()
	cfg := sessionConfig.New(log)

	webMetrics := web.New("api")
	dbMetric := database.New("api")
	metrics.New(log, append(webMetrics.List(), dbMetric.List()...))

	pgPool, err := pool.New(dbMetric, cfg.Postgres.String())
	if err != nil {
		log.Fatal(ctx, "can't init postgres connection pool: %s", err)
	}
	defer pgPool.Close()

	var chConnection driver.Conn
	// Skip ClickHouse initialization if URL is not configured
	if cfg.Clickhouse.URL != "" {
		chConnection, err = clickhouse.NewConnection(cfg.Clickhouse)
		if err != nil {
			log.Fatal(ctx, "can't init clickhouse connection: %s", err)
		}
	} else {
		log.Info(ctx, "ClickHouse is disabled (no URL configured)")
	}

	objStore, err := store.NewStore(&cfg.ObjectsConfig)
	if err != nil {
		log.Fatal(ctx, "can't init object storage: %s", err)
	}

	projectsSvc := projects.New(log, pgPool, nil, dbMetric)

	canvasesSvc, err := canvas.New(log, pgPool, dbMetric)
	if err != nil {
		log.Fatal(ctx, "can't init project service: %s", err)
	}

	services, err := apiService.NewServiceBuilder(log, cfg, webMetrics, pgPool, chConnection, objStore, projectsSvc, canvasesSvc)
	if err != nil {
		log.Fatal(ctx, "can't init services and handlers: %s", err)
	}

	tenants := tenant.New(pgPool)
	prefix := api.NoPrefix

	middlewares, err := middleware.NewMiddlewareBuilder(log, cfg.JWTSecret, cfg.GoInterserviceAPIKey, &cfg.HTTP, &cfg.RateLimiter, prefix, pgPool, dbMetric, services.Handlers(), tenants, projectsSvc, nil, nil)
	if err != nil {
		log.Fatal(ctx, "can't init middlewares: %s", err)
	}

	router, err := api.NewRouter(log, &cfg.HTTP, prefix, services.Handlers(), middlewares.Middlewares())
	if err != nil {
		log.Fatal(ctx, "failed while creating router: %s", err)
	}

	server.Run(ctx, log, &cfg.HTTP, router)
}
