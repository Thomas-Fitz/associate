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

	mcpserver "github.com/fitz/associate/internal/mcp"
	"github.com/fitz/associate/internal/neo4j"
)

func main() {
	// Parse flags
	httpMode := flag.Bool("http", false, "Run as HTTP server (default: stdio for MCP)")
	port := flag.Int("port", 8080, "HTTP port to listen on (only used with -http)")
	waitForDB := flag.Bool("wait", true, "Wait for Neo4j to be available (with retries)")
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

	// Connect to Neo4j
	cfg := neo4j.ConfigFromEnv()
	logger.Info("connecting to Neo4j", "uri", cfg.URI, "database", cfg.Database)

	var client *neo4j.Client
	var err error

	if *waitForDB {
		// Use retry logic - useful when starting before Neo4j is ready
		logger.Info("waiting for Neo4j to be available...")
		client, err = neo4j.NewClientWithRetry(ctx, cfg, nil)
	} else {
		// Direct connection - fails immediately if Neo4j unavailable
		client, err = neo4j.NewClient(ctx, cfg)
	}
	if err != nil {
		logger.Error("failed to connect to Neo4j", "error", err)
		os.Exit(1)
	}
	defer client.Close(ctx)

	logger.Info("connected to Neo4j")

	// Create repository and MCP server
	repo := neo4j.NewRepository(client)
	server := mcpserver.NewServer(repo, logger)

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
