package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"searchtags/internal/api"
	"searchtags/internal/tags"
	"searchtags/internal/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{getenv("CLICKHOUSE_ADDR", "localhost:9000")},
		Auth: clickhouse.Auth{
			Database: getenv("CLICKHOUSE_DB", "search"),
			Username: getenv("CLICKHOUSE_USER", "app"),
			Password: getenv("CLICKHOUSE_PASSWORD", "app"),
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("open clickhouse: %w", err)
	}
	defer conn.Close()

	// Проверяем связь при старте: если базы нет, лучше упасть сразу и понятно.
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = conn.Ping(pingCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("ping clickhouse: %w", err)
	}

	store := tags.NewClickHouseStore(conn, 3*time.Second)

	mux := http.NewServeMux()
	api.NewHandler(store, logger).Register(mux)
	page, err := web.NewHandler(store, logger)
	if err != nil {
		return fmt.Errorf("init web handler: %w", err)
	}
	page.Register(mux)

	srv := &http.Server{
		Addr:              ":" + getenv("PORT", "8080"),
		Handler:           api.LogRequests(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Ctrl+C или SIGTERM запустят аккуратную остановку.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server started", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
