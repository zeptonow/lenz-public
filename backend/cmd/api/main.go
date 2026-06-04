package main

import (
	"context"

	sessionConfig "openreplay/backend/internal/config/api"
	apiService "openreplay/backend/pkg/api"
	"openreplay/backend/pkg/canvas"
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
	/*
		Replay-only build: ClickHouse/analytics disabled.
		Original imports kept here for later re-enable.

		"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
		"openreplay/backend/pkg/db/clickhouse"
	*/)

func main() {
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

	// Replay-only build: ClickHouse disabled (kept for later).
	// var chConnection driver.Conn
	// if cfg.Clickhouse.Enabled {
	// 	chConnection, err = clickhouse.NewConnection(cfg.Clickhouse)
	// 	if err != nil {
	// 		log.Fatal(ctx, "can't init clickhouse connection: %s", err)
	// 	}
	// 	log.Info(ctx, "ClickHouse connection established")
	// } else {
	// 	log.Info(ctx, "ClickHouse is disabled, analytics features will be unavailable")
	// 	chConnection = nil
	// }
	// var chConnection any = nil

	objStore, err := store.NewStore(&cfg.ObjectsConfig)
	if err != nil {
		log.Fatal(ctx, "can't init object storage: %s", err)
	}

	projects := projects.New(log, pgPool, nil, dbMetric)
	if err != nil {
		log.Fatal(ctx, "can't init project service: %s", err)
	}

	canvases, err := canvas.New(log, pgPool, dbMetric)
	if err != nil {
		log.Fatal(ctx, "can't init project service: %s", err)
	}

	services, err := apiService.NewServiceBuilder(log, cfg, webMetrics, pgPool, nil, objStore, projects, canvases)
	if err != nil {
		log.Fatal(ctx, "can't init services and handlers: %s", err)
	}

	tenants := tenant.New(pgPool)
	prefix := api.NoPrefix

	middlewares, err := middleware.NewMiddlewareBuilder(log, cfg.JWTSecret, cfg.GoInterserviceAPIKey, &cfg.HTTP, &cfg.RateLimiter, prefix, pgPool, dbMetric, services.Handlers(), tenants, projects, nil, nil)
	if err != nil {
		log.Fatal(ctx, "can't init middlewares: %s", err)
	}

	router, err := api.NewRouter(log, &cfg.HTTP, prefix, services.Handlers(), middlewares.Middlewares())
	if err != nil {
		log.Fatal(ctx, "failed while creating router: %s", err)
	}

	server.Run(ctx, log, &cfg.HTTP, router)
}
