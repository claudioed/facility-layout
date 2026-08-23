//go:build integration

package postgres_test

import "github.com/jackc/pgx/v5/pgxpool"

// pgxPool is a local alias so the integration helpers can name the pool
// type without every test file importing pgxpool directly.
type pgxPool = pgxpool.Pool
