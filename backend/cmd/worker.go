package cmd

import (
	"context"
	"fmt"
	"openreplay/backend/internal/config/http"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/server"
	"openreplay/backend/pkg/server/api"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(workerCmd)
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start all OpenReplay backend services in a single process",
	Run: func(cmd *cobra.Command, args []string) {
		runWorker()
	},
}

func runWorker() {
	// Core services for session recording/replay (replay-only build).
	services := []struct {
		name string
		fn   func()
	}{
		{"health", runHealth}, // Port 8080 - health endpoint
		//{"http", runHTTP},         // Ingestion HTTP API
		{"sink", runSink},         // Processes incoming messages, writes to FS
		{"ender", runEnder},       // Detects session timeouts, produces SessionEnd
		{"db", runDB},             // Persists session metadata (PG); analytics optional
		{"storage", runStorage},   // Consumes SessionEnd, uploads to S3
		{"assets", runAssets},     // Asset caching
		{"images", runImages},     // Image processing
		{"canvases", runCanvases}, // Canvas recording

		// Replay-only build: disabled services kept for later.
		// {"heuristics", runHeuristics}, // Issue detection / analytics
	}

	for _, svc := range services {
		s := svc
		go func() {
			fmt.Printf("[%s] Starting service...\n", s.name)
			s.fn()
		}()
	}

	fmt.Println("All services started. Waiting for shutdown signal...")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigchan
	fmt.Printf("Caught signal %v: shutting down all services\n", sig)
	os.Exit(0)
}

func runHealth() {
	ctx := context.Background()
	log := logger.New()
	cfg := http.New(log)

	router, err := api.NewRouter(log, &cfg.HTTP, api.NoPrefix, nil, nil)
	if err != nil {
		log.Fatal(ctx, "failed while creating router: %s", err)
	}
	server.Run(ctx, log, &cfg.HTTP, router)
}
