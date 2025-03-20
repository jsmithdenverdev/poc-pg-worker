package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/jackc/pgx/v5/pgxpool"
)

type config struct {
	DatabaseURL string `env:"DATABASE_URL"`
	ServerPort  string `env:"SERVER_PORT"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration from environment
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	// Create connection pool
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}
	defer pool.Close()

	// Set up routes
	svr := newServer(pool)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", cfg.ServerPort),
		Handler: svr,
	}

	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()
	// Start the worker
	workerFn := worker(pool, logger, "tasks_channel")
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := workerFn(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "worker error: %s\n", err)
		}
	}()

	wg.Wait()
	return nil
}
