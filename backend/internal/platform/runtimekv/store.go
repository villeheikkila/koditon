package runtimekv

import (
	"context"
	"errors"
	"fmt"
	"time"

	"koditon/internal/db"

	"github.com/jackc/pgx/v5"
)

type Store struct {
	queries *db.Queries
}

func New(queries *db.Queries) *Store {
	return &Store{queries: queries}
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	payload, err := s.queries.GetRuntimeKV(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get runtime kv: %w", err)
	}
	return payload, nil
}

func (s *Store) Set(ctx context.Context, key string, payload []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Hour
	}
	err := s.queries.UpsertRuntimeKV(ctx, db.UpsertRuntimeKVParams{
		KvKey:     key,
		KvValue:   payload,
		ExpiresAt: time.Now().Add(ttl),
	})
	if err != nil {
		return fmt.Errorf("upsert runtime kv: %w", err)
	}
	return nil
}
