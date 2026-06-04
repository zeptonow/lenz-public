package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"openreplay/backend/internal/assets"
	"openreplay/backend/internal/assets/cacher"
	config "openreplay/backend/internal/config/assets"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics"
	assetsMetrics "openreplay/backend/pkg/metrics/assets"
	"openreplay/backend/pkg/objectstorage/store"
	"openreplay/backend/pkg/queue"
	"openreplay/backend/pkg/queue/types"
)

func init() {
	rootCmd.AddCommand(assetsCmd)
}

var assetsCmd = &cobra.Command{
	Use:   "assets",
	Short: "Start the Assets caching service",
	Run: func(cmd *cobra.Command, args []string) {
		runAssets()
	},
}

func runAssets() {
	ctx := context.Background()
	log := logger.New()
	cfg := config.New(log)

	assetMetrics := assetsMetrics.New("assets")
	metrics.New(log, assetMetrics.List())

	// Configure redisstream before creating consumer
	queue.SetRedisConfig(cfg.Redis.ConnectionURL, int64(cfg.Redis.MaxLength))

	objStore, err := store.NewStore(&cfg.ObjectsConfig)
	if err != nil {
		log.Fatal(ctx, "can't init object storage: %s", err)
	}
	cacherSvc, err := cacher.NewCacher(cfg, objStore, assetMetrics)
	if err != nil {
		log.Fatal(ctx, "can't init cacher: %s", err)
	}

	msgHandler := func(msg messages.Message) {
		switch m := msg.(type) {
		case *messages.AssetCache:
			cacherSvc.CacheURL(m.SessionID(), m.URL)
			assetMetrics.IncreaseProcessesSessions()
		case *messages.JSException:
			sourceList, err := assets.ExtractJSExceptionSources(&m.Payload)
			if err != nil {
				log.Error(ctx, "Error on source extraction: %s", err)
				return
			}
			for _, source := range sourceList {
				cacherSvc.CacheJSFile(source)
			}
		}
	}

	msgConsumer, err := queue.NewConsumer(
		log,
		cfg.GroupCache,
		[]string{cfg.TopicCache},
		messages.NewMessageIterator(log, msgHandler, []int{messages.MsgAssetCache, messages.MsgJSException}, true),
		true,
		cfg.MessageSizeLimit,
		nil,
		types.NoReadBackGap,
	)
	if err != nil {
		log.Fatal(ctx, "can't init message consumer: %s", err)
	}

	log.Info(ctx, "Cacher service started")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	tick := time.Tick(20 * time.Minute)
	for {
		select {
		case sig := <-sigchan:
			log.Error(ctx, "Caught signal %v: terminating", sig)
			cacherSvc.Stop()
			msgConsumer.Close()
			os.Exit(0)
		case err := <-cacherSvc.Errors:
			log.Error(ctx, "Error while caching: %s", err)
		case <-tick:
			cacherSvc.UpdateTimeouts()
		default:
			if !cacherSvc.CanCache() {
				continue
			}
			if err := msgConsumer.ConsumeNext(); err != nil {
				log.Fatal(ctx, "Error on consumption: %v", err)
			}
		}
	}
}
