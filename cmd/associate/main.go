package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Thomas-Fitz/associate/internal/graph"
	mcpserver "github.com/Thomas-Fitz/associate/internal/mcp"
)

func main() {
	// Parse flags
	httpMode := flag.Bool("http", false, "Run as HTTP server (default: stdio for MCP)")
	port := flag.Int("port", 8080, "HTTP port to listen on (only used with -http)")
	waitForDB := flag.Bool("wait", true, "Wait for PostgreSQL/AGE to be available (with retries)")
	flag.Parse()

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("shutting down...")
		cancel()
	}()

	// Connect to PostgreSQL/AGE
	cfg := graph.ConfigFromEnv()
	logger.Info("connecting to PostgreSQL/AGE", "host", cfg.Host, "port", cfg.Port, "database", cfg.Database)

	var client *graph.Client
	var err error

	if *waitForDB {
		// Use retry logic - useful when starting before PostgreSQL is ready
		logger.Info("waiting for PostgreSQL/AGE to be available...")
		client, err = graph.NewClientWithRetry(ctx, cfg, nil)
	} else {
		// Direct connection - fails immediately if PostgreSQL unavailable
		client, err = graph.NewClient(ctx, cfg)
	}
	if err != nil {
		logger.Error("failed to connect to PostgreSQL/AGE", "error", err)
		os.Exit(1)
	}
	defer client.Close(ctx)

	logger.Info("connected to PostgreSQL/AGE")

	// Create repositories and MCP server
	repo := graph.NewRepository(client)
	planRepo := graph.NewPlanRepository(client)
	taskRepo := graph.NewTaskRepository(client)
	server := mcpserver.NewServer(repo, planRepo, taskRepo, logger)

	if *httpMode {
		// Run as HTTP server
		addr := fmt.Sprintf(":%d", *port)
		httpServer := &http.Server{
			Addr:              addr,
			Handler:           server.HTTPHandler(),
			ReadHeaderTimeout: 10 * time.Second,
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			_ = httpServer.Shutdown(shutdownCtx)
		}()

		logger.Info("starting HTTP server", "addr", addr)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	} else {
		// Run as stdio server
		logger.Info("starting MCP server on stdio")
		if err := server.Run(ctx); err != nil {
			logger.Error("MCP server error", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("server stopped")
}
