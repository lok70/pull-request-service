// Package repository реализует работу с psql - сохранение и чтение команд, пользователей и pr.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres инкапсулирует пул подключений к PostgreSQL.
type Postgres struct {
	Pool *pgxpool.Pool
}

// NewPostgres создаёт и инициализирует пул подключений к PostgreSQL по переданному DSN и возвращает обёртку Postgres.
func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	return &Postgres{Pool: pool}, nil
}
