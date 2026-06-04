package cmd

import (
	"context"

	"github.com/spf13/cobra"

	config "openreplay/backend/internal/config/db"
	"openreplay/backend/internal/db"
	"openreplay/backend/internal/db/datasaver"
	"openreplay/backend/pkg/canvas"
	"openreplay/backend/pkg/db/postgres/pool"
	"openreplay/backend/pkg/db/redis"
	"openreplay/backend/pkg/issues"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/memory"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics"
	"openreplay/backend/pkg/metrics/database"
	"openreplay/backend/pkg/projects"
	"openreplay/backend/pkg/queue"
	"openreplay/backend/pkg/queue/types"
	"openreplay/backend/pkg/sessions"
	"openreplay/backend/pkg/tags"
	"openreplay/backend/pkg/terminator"
)

func init() {
	rootCmd.AddCommand(dbCmd)
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Start the DB consumer service",
	Run: func(cmd *cobra.Command, args []string) {
		runDB()
	},
}

func runDB() {
	ctx := context.Background()
	log := logger.New()
	cfg := config.New(log)

	dbMetric := database.New("db")
	metrics.New(log, dbMetric.List())

	// Configure redisstream before creating consumer
	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

	pgConn, err := pool.New(dbMetric, cfg.Postgres.String())
	if err != nil {
		log.Fatal(ctx, "can't init postgres connection: %s", err)
	}
	defer pgConn.Close()

	// Replay-only build: ClickHouse/analytics disabled (kept for later).
	// var chConn driver.Conn
	// if cfg.Clickhouse.URL != "" {
	// 	chConn, err = clickhouse.NewConnection(cfg.Clickhouse)
	// 	if err != nil {
	// 		log.Fatal(ctx, "can't init clickhouse connection: %s", err)
	// 	}
	// } else {
	// 	log.Info(ctx, "ClickHouse is disabled (no URL configured)")
	// }

	redisConn, err := redis.New(&cfg.Redis)
	if err != nil {
		log.Warn(ctx, "can't init redis connection: %s", err)
	}
	defer redisConn.Close()

	issuesManager, err := issues.New(log, redisConn)
	if err != nil {
		log.Fatal(ctx, "can't init issues keeper: %s", err)
	}

	// Replay-only build: disable ClickHouse connector.
	// var chConnector clickhouse.Connector

	projManager := projects.New(log, pgConn, redisConn, dbMetric)
	sessManager := sessions.New(log, pgConn, projManager, redisConn, dbMetric)
	tagsManager := tags.New(log, pgConn)

	canvases, err := canvas.New(log, pgConn, dbMetric)
	if err != nil {
		log.Fatal(ctx, "can't init project service: %s", err)
	}

	// Replay-only build: disable ClickHouse-backed users + SDK saver.
	// var users sdk.Users

	saver := datasaver.New(log, cfg, nil, sessManager, issuesManager, tagsManager, canvases, nil)

	msgFilter := []int{
		messages.MsgMetadata, messages.MsgIssueEvent, messages.MsgSessionStart, messages.MsgSessionEnd,
		messages.MsgUserID, messages.MsgUserAnonymousID, messages.MsgPerformanceTrackAggr,
		messages.MsgJSException, messages.MsgResourceTiming, messages.MsgCustomEvent, messages.MsgCustomIssue,
		messages.MsgNetworkRequest, messages.MsgGraphQL, messages.MsgStateAction, messages.MsgMouseClick,
		messages.MsgMouseClickDeprecated, messages.MsgSetPageLocation, messages.MsgSetPageLocationDeprecated,
		messages.MsgPageLoadTiming, messages.MsgPageRenderTiming,
		messages.MsgPageEvent, messages.MsgPageEventDeprecated, messages.MsgMouseThrashing, messages.MsgInputChange,
		messages.MsgUnbindNodes, messages.MsgTagTrigger, messages.MsgIncident, messages.MsgCanvasNode,
		messages.MsgMobileSessionStart, messages.MsgMobileSessionEnd, messages.MsgMobileUserID, messages.MsgMobileUserAnonymousID,
		messages.MsgMobileMetadata, messages.MsgMobileEvent, messages.MsgMobileNetworkCall,
		messages.MsgMobileClickEvent, messages.MsgMobileSwipeEvent, messages.MsgMobileInputEvent,
		messages.MsgMobileCrash, messages.MsgMobileIssueEvent,
	}

	consumer, err := queue.NewConsumer(
		log,
		cfg.GroupDB,
		[]string{cfg.TopicRawWeb, cfg.TopicRawMobile /* cfg.TopicAnalytics */},
		messages.NewMessageIterator(log, saver.Handle, msgFilter, true),
		false,
		cfg.MessageSizeLimit,
		nil,
		types.NoReadBackGap,
	)
	if err != nil {
		log.Fatal(ctx, "can't init message consumer: %s", err)
	}

	// Replay-only build: disable SDK saver.

	memoryManager, err := memory.NewManager(log, cfg.MemoryLimitMB, cfg.MaxMemoryUsage)
	if err != nil {
		log.Fatal(ctx, "can't init memory manager: %s", err)
	}

	service := db.New(log, cfg, consumer, saver, memoryManager, sessManager)
	log.Info(ctx, "Db service started")
	terminator.Wait(log, service)
}
