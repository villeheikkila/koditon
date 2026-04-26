package runtimekv

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `
		SELECT kv_value
		FROM runtime.kv_store
		WHERE kv_key = $1
		  AND expires_at > now()
	`, key).Scan(&payload)
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
	_, err := s.pool.Exec(ctx, `
		INSERT INTO runtime.kv_store (
			kv_key,
			kv_value,
			expires_at
		) VALUES (
			$1,
			$2,
			$3
		)
		ON CONFLICT (kv_key) DO UPDATE SET
			kv_value = EXCLUDED.kv_value,
			expires_at = EXCLUDED.expires_at,
			updated_at = now()
	`, key, payload, time.Now().Add(ttl))
	if err != nil {
		return fmt.Errorf("upsert runtime kv: %w", err)
	}
	return nil
}
