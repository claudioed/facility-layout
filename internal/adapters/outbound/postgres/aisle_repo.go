package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// AisleRepo is a pgxpool-backed implementation of ports.AisleRepo.
type AisleRepo struct {
	pool *pgxpool.Pool
}

// NewAisleRepo builds an AisleRepo over pool.
func NewAisleRepo(pool *pgxpool.Pool) *AisleRepo {
	return &AisleRepo{pool: pool}
}

// Save upserts the aisle.
func (r *AisleRepo) Save(ctx context.Context, a *aisle.Aisle) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO aisles (id, zone_id, aisle_code, sequence_hint, direction, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			sequence_hint = EXCLUDED.sequence_hint,
			direction = EXCLUDED.direction,
			status = EXCLUDED.status
	`, a.ID(), a.ZoneID(), a.AisleCode(), a.SequenceHint(), string(a.Direction()), string(a.Status()))
	return err
}

// FindByID returns the aisle, or (nil, nil) when it does not exist.
func (r *AisleRepo) FindByID(ctx context.Context, id string) (*aisle.Aisle, error) {
	var zoneID, aisleCode, direction, status string
	var sequenceHint int
	err := r.pool.QueryRow(ctx, `
		SELECT zone_id, aisle_code, sequence_hint, direction, status FROM aisles WHERE id = $1
	`, id).Scan(&zoneID, &aisleCode, &sequenceHint, &direction, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rehydrateAisle(zoneID, aisleCode, sequenceHint, direction, status)
}

// ListByZone returns every aisle in a zone, in walk order.
func (r *AisleRepo) ListByZone(ctx context.Context, zoneID string) ([]*aisle.Aisle, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT zone_id, aisle_code, sequence_hint, direction, status
		FROM aisles WHERE zone_id = $1 ORDER BY sequence_hint, aisle_code
	`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*aisle.Aisle, 0)
	for rows.Next() {
		var zone, aisleCode, direction, status string
		var sequenceHint int
		if err := rows.Scan(&zone, &aisleCode, &sequenceHint, &direction, &status); err != nil {
			return nil, err
		}
		a, err := rehydrateAisle(zone, aisleCode, sequenceHint, direction, status)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func rehydrateAisle(zoneID, aisleCode string, sequenceHint int, direction, status string) (*aisle.Aisle, error) {
	dir, err := shared.ParseDirection(direction)
	if err != nil {
		return nil, err
	}
	st, err := shared.ParseStatus(status)
	if err != nil {
		return nil, err
	}
	return aisle.RehydrateAisle(zoneID, aisleCode, sequenceHint, dir, st), nil
}
