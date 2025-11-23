package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type txKey struct{}

// TransactionManager управляет транзакциями.
type TransactionManager struct {
	db *Postgres
}

// NewTransactionManager создаёт новый менеджер.
func NewTransactionManager(db *Postgres) *TransactionManager {
	return &TransactionManager{db: db}
}

// RunInTransaction выполняет функцию fn внутри транзакции.
func (tm *TransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := tm.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	// Кладём транзакцию в контекст
	ctx = context.WithValue(ctx, txKey{}, tx)

	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback error: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// DBTX описывает общий интерфейс для *pgxpool.Pool и pgx.Tx.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

// GetQueryExecutor возвращает транзакцию из контекста, если она есть,
// или пул соединений, если транзакции нет.
func (p *Postgres) GetQueryExecutor(ctx context.Context) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return p.Pool
}
