package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{getenv("CLICKHOUSE_ADDR", "localhost:9000")},
		Auth: clickhouse.Auth{
			Database: "search",
			Username: getenv("CLICKHOUSE_USER", "app"),
			Password: getenv("CLICKHOUSE_PASSWORD", "app"),
		},
	})
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}

	var version string
	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		log.Fatalf("query: %v", err)
	}
	fmt.Println("connected, ClickHouse version:", version)
}
