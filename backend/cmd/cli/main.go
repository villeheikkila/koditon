package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/platform/config"
)

func main() {
	if err := newRootCommand(context.Background(), os.Stdout, os.Stderr, os.Getenv).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func setup(ctx context.Context) (*pgxpool.Pool, config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, config.Config{}, fmt.Errorf("load config: %w", err)
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, config.Config{}, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, config.Config{}, fmt.Errorf("ping database: %w", err)
	}
	return pool, cfg, nil
}
