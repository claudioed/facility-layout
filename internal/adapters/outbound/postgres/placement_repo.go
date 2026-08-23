package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// LocationTypeRepo is a pgxpool-backed implementation of ports.LocationTypeRepo.
type LocationTypeRepo struct {
	pool *pgxpool.Pool
}

// NewLocationTypeRepo builds a LocationTypeRepo over pool.
func NewLocationTypeRepo(pool *pgxpool.Pool) *LocationTypeRepo {
	return &LocationTypeRepo{pool: pool}
}

// Save upserts the location type.
func (r *LocationTypeRepo) Save(ctx context.Context, t placement.LocationType) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO location_types (name, default_max_weight_kg, default_max_volume_m3)
		VALUES ($1, $2, $3)
		ON CONFLICT (name) DO UPDATE SET
			default_max_weight_kg = EXCLUDED.default_max_weight_kg,
			default_max_volume_m3 = EXCLUDED.default_max_volume_m3
	`, t.Name(), t.DefaultCapacity().MaxWeightKg(), t.DefaultCapacity().MaxVolumeM3())
	return err
}

// FindByName returns the location type, or (nil, nil) when it does not exist.
func (r *LocationTypeRepo) FindByName(ctx context.Context, name string) (*placement.LocationType, error) {
	var weight, volume float64
	err := r.pool.QueryRow(ctx, `
		SELECT default_max_weight_kg, default_max_volume_m3 FROM location_types WHERE name = $1
	`, name).Scan(&weight, &volume)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	capacity, err := shared.NewCapacity(weight, volume)
	if err != nil {
		return nil, err
	}
	t := placement.RehydrateLocationType(name, capacity)
	return &t, nil
}

// List returns every location type, ordered by name.
func (r *LocationTypeRepo) List(ctx context.Context) ([]placement.LocationType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT name, default_max_weight_kg, default_max_volume_m3 FROM location_types ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]placement.LocationType, 0)
	for rows.Next() {
		var name string
		var weight, volume float64
		if err := rows.Scan(&name, &weight, &volume); err != nil {
			return nil, err
		}
		capacity, err := shared.NewCapacity(weight, volume)
		if err != nil {
			return nil, err
		}
		out = append(out, placement.RehydrateLocationType(name, capacity))
	}
	return out, rows.Err()
}

// PlacementRuleRepo is a pgxpool-backed implementation of ports.PlacementRuleRepo.
type PlacementRuleRepo struct {
	pool *pgxpool.Pool
}

// NewPlacementRuleRepo builds a PlacementRuleRepo over pool.
func NewPlacementRuleRepo(pool *pgxpool.Pool) *PlacementRuleRepo {
	return &PlacementRuleRepo{pool: pool}
}

// Save upserts the placement rule. An unconstrained predicate dimension is
// stored as SQL NULL, matching the domain's "wildcard" semantics.
func (r *PlacementRuleRepo) Save(ctx context.Context, rule placement.PlacementRule) error {
	predicate := rule.Predicate()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO placement_rules (id, location_type, effect, zone_code, temperature_class, hazmat)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			location_type = EXCLUDED.location_type,
			effect = EXCLUDED.effect,
			zone_code = EXCLUDED.zone_code,
			temperature_class = EXCLUDED.temperature_class,
			hazmat = EXCLUDED.hazmat
	`, rule.ID(), rule.LocationType(), string(rule.Effect()),
		nullableString(predicate.ZoneCode()),
		nullableString(string(predicate.TemperatureClass())),
		predicate.Hazmat())
	return err
}

// FindByID returns the rule, or (nil, nil) when it does not exist.
func (r *PlacementRuleRepo) FindByID(ctx context.Context, id string) (*placement.PlacementRule, error) {
	var locationType, effect string
	var zoneCode, temperatureClass *string
	var hazmat *bool
	err := r.pool.QueryRow(ctx, `
		SELECT location_type, effect, zone_code, temperature_class, hazmat
		FROM placement_rules WHERE id = $1
	`, id).Scan(&locationType, &effect, &zoneCode, &temperatureClass, &hazmat)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rule, err := rehydrateRule(id, locationType, effect, zoneCode, temperatureClass, hazmat)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// List returns every rule, ordered by id.
func (r *PlacementRuleRepo) List(ctx context.Context) ([]placement.PlacementRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, location_type, effect, zone_code, temperature_class, hazmat
		FROM placement_rules ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]placement.PlacementRule, 0)
	for rows.Next() {
		var id, locationType, effect string
		var zoneCode, temperatureClass *string
		var hazmat *bool
		if err := rows.Scan(&id, &locationType, &effect, &zoneCode, &temperatureClass, &hazmat); err != nil {
			return nil, err
		}
		rule, err := rehydrateRule(id, locationType, effect, zoneCode, temperatureClass, hazmat)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

func rehydrateRule(id, locationType, effect string, zoneCode, temperatureClass *string, hazmat *bool) (placement.PlacementRule, error) {
	eff, err := placement.ParseEffect(effect)
	if err != nil {
		return placement.PlacementRule{}, err
	}
	predicate, err := placement.NewZonePredicate(
		derefString(zoneCode),
		shared.TemperatureClass(derefString(temperatureClass)),
		hazmat,
	)
	if err != nil {
		return placement.PlacementRule{}, err
	}
	return placement.RehydratePlacementRule(id, locationType, eff, predicate), nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
