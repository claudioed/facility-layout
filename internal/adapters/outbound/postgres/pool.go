// Package postgres provides pgxpool-backed implementations of every
// outbound port, plus a golang-migrate runner for the SQL migrations in
// /migrations.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens a connection pool against databaseURL.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, databaseURL)
}
