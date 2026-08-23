package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/slot"
)

// SlotRepo is a pgxpool-backed implementation of ports.SlotRepo. The
// LocationCode's seven segments are stored as real columns alongside the
// canonical string form, so the zone-grid read model can be answered with
// an indexed query rather than by parsing codes at read time.
type SlotRepo struct {
	pool *pgxpool.Pool
}

// NewSlotRepo builds a SlotRepo over pool.
func NewSlotRepo(pool *pgxpool.Pool) *SlotRepo {
	return &SlotRepo{pool: pool}
}

const slotColumns = `code, site_segment, area_segment, zone_segment, aisle_segment,
	bay_segment, level_segment, position_segment, location_type, max_weight_kg, max_volume_m3, status`

// Save upserts the slot.
func (r *SlotRepo) Save(ctx context.Context, s *slot.LocationSlot) error {
	code := s.Code()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO location_slots (
			code, site_segment, area_segment, zone_segment, aisle_segment,
			bay_segment, level_segment, position_segment,
			zone_id, aisle_id, location_type, max_weight_kg, max_volume_m3, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (code) DO UPDATE SET
			location_type = EXCLUDED.location_type,
			max_weight_kg = EXCLUDED.max_weight_kg,
			max_volume_m3 = EXCLUDED.max_volume_m3,
			status = EXCLUDED.status
	`,
		code.String(), code.Site(), code.Area(), code.Zone(), code.Aisle(),
		code.Bay(), code.Level(), code.Position(),
		code.ZoneID(), code.AisleID(), s.LocationType(),
		s.Capacity().MaxWeightKg(), s.Capacity().MaxVolumeM3(), string(s.Status()))
	return err
}

// FindByCode returns the slot, or (nil, nil) when it does not exist.
func (r *SlotRepo) FindByCode(ctx context.Context, code shared.LocationCode) (*slot.LocationSlot, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+slotColumns+` FROM location_slots WHERE code = $1`, code.String())

	s, err := scanSlot(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ListByAisle returns every slot in an aisle, ordered bay -> level -> position.
func (r *SlotRepo) ListByAisle(ctx context.Context, aisleID string) ([]*slot.LocationSlot, error) {
	return r.list(ctx, `SELECT `+slotColumns+` FROM location_slots WHERE aisle_id = $1
		ORDER BY bay_segment, level_segment, position_segment`, aisleID)
}

// ListByZone returns every slot in a zone, ordered aisle -> bay -> level -> position.
func (r *SlotRepo) ListByZone(ctx context.Context, zoneID string) ([]*slot.LocationSlot, error) {
	return r.list(ctx, `SELECT `+slotColumns+` FROM location_slots WHERE zone_id = $1
		ORDER BY aisle_segment, bay_segment, level_segment, position_segment`, zoneID)
}

func (r *SlotRepo) list(ctx context.Context, query, arg string) ([]*slot.LocationSlot, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*slot.LocationSlot, 0)
	for rows.Next() {
		s, err := scanSlot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// scanner is the shared shape of pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanSlot(row scanner) (*slot.LocationSlot, error) {
	var raw, site, area, zoneSeg, aisleSeg, bay, level, position, locationType, status string
	var maxWeightKg, maxVolumeM3 float64

	if err := row.Scan(&raw, &site, &area, &zoneSeg, &aisleSeg, &bay, &level, &position,
		&locationType, &maxWeightKg, &maxVolumeM3, &status); err != nil {
		return nil, err
	}

	code, err := shared.NewLocationCode(site, area, zoneSeg, aisleSeg, bay, level, position)
	if err != nil {
		return nil, err
	}
	capacity, err := shared.NewCapacity(maxWeightKg, maxVolumeM3)
	if err != nil {
		return nil, err
	}
	st, err := shared.ParseStatus(status)
	if err != nil {
		return nil, err
	}
	return slot.RehydrateLocationSlot(code, locationType, capacity, st), nil
}
