package cmd

import (
	"context"

	"github.com/spf13/cobra"

	config "openreplay/backend/internal/config/heuristics"
	"openreplay/backend/pkg/handlers"
	"openreplay/backend/pkg/handlers/custom"
	"openreplay/backend/pkg/handlers/mobile"
	"openreplay/backend/pkg/handlers/web"
	"openreplay/backend/pkg/heuristics"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/memory"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics"
	heuristicsMetrics "openreplay/backend/pkg/metrics/heuristics"
	"openreplay/backend/pkg/queue"
	"openreplay/backend/pkg/queue/types"
	"openreplay/backend/pkg/terminator"
)

func init() {
	rootCmd.AddCommand(heuristicsCmd)
}

var heuristicsCmd = &cobra.Command{
	Use:   "heuristics",
	Short: "Start Heuristics service",
	Run: func(cmd *cobra.Command, args []string) {
		runHeuristics()
	},
}

func runHeuristics() {
	ctx := context.Background()
	log := logger.New()
	cfg := config.New(log)

	heuristicsMetric := heuristicsMetrics.New("heuristics")
	metrics.New(log, heuristicsMetric.List())

	handlersFabric := func() []handlers.MessageProcessor {
		return []handlers.MessageProcessor{
			custom.NewPageEventBuilder(),
			web.NewDeadClickDetector(),
			&web.ClickRageDetector{},
			&web.CpuIssueDetector{},
			&web.MemoryIssueDetector{},
			&web.NetworkIssueDetector{},
			&web.PerformanceAggregator{},
			web.NewAppCrashDetector(),
			&mobile.TapRageDetector{},
			mobile.NewViewComponentDurations(),
		}
	}

	// Configure redisstream before creating producer/consumer
	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

	events := heuristics.NewEvents(log, handlersFabric)
	producer := queue.NewProducer(cfg.MessageSizeLimit, true, nil)
	consumer, err := queue.NewConsumer(
		log,
		cfg.GroupHeuristics,
		[]string{cfg.TopicRawWeb, cfg.TopicRawMobile},
		messages.NewMessageIterator(log, events.HandleMessage, nil, true),
		false,
		cfg.MessageSizeLimit,
		nil,
		types.NoReadBackGap,
	)
	if err != nil {
		log.Fatal(ctx, "can't init message consumer: %s", err)
	}

	memoryManager, err := memory.NewManager(log, cfg.MemoryLimitMB, cfg.MaxMemoryUsage)
	if err != nil {
		log.Fatal(ctx, "can't init memory manager: %s", err)
	}

	service := heuristics.New(log, cfg, producer, consumer, events, memoryManager, heuristicsMetric)
	log.Info(ctx, "Heuristics service started")
	terminator.Wait(log, service)
}
