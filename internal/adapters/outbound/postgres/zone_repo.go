package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

// ZoneRepo is a pgxpool-backed implementation of ports.ZoneRepo.
type ZoneRepo struct {
	pool *pgxpool.Pool
}

// NewZoneRepo builds a ZoneRepo over pool.
func NewZoneRepo(pool *pgxpool.Pool) *ZoneRepo {
	return &ZoneRepo{pool: pool}
}

// Save upserts the zone.
func (r *ZoneRepo) Save(ctx context.Context, z *zone.Zone) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO zones (id, site_code, area_code, zone_code, temperature_class, hazmat, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			temperature_class = EXCLUDED.temperature_class,
			hazmat = EXCLUDED.hazmat,
			status = EXCLUDED.status
	`, z.ID(), z.SiteCode(), z.AreaCode(), z.ZoneCode(), string(z.TemperatureClass()), z.Hazmat(), string(z.Status()))
	return err
}

// FindByID returns the zone, or (nil, nil) when it does not exist.
func (r *ZoneRepo) FindByID(ctx context.Context, id string) (*zone.Zone, error) {
	var siteCode, areaCode, zoneCode, temperatureClass, status string
	var hazmat bool
	err := r.pool.QueryRow(ctx, `
		SELECT site_code, area_code, zone_code, temperature_class, hazmat, status
		FROM zones WHERE id = $1
	`, id).Scan(&siteCode, &areaCode, &zoneCode, &temperatureClass, &hazmat, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rehydrateZone(siteCode, areaCode, zoneCode, temperatureClass, hazmat, status)
}

// ListBySite returns every zone in a site, ordered by id.
func (r *ZoneRepo) ListBySite(ctx context.Context, siteCode string) ([]*zone.Zone, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT site_code, area_code, zone_code, temperature_class, hazmat, status
		FROM zones WHERE site_code = $1 ORDER BY id
	`, siteCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*zone.Zone, 0)
	for rows.Next() {
		var site, areaCode, zoneCode, temperatureClass, status string
		var hazmat bool
		if err := rows.Scan(&site, &areaCode, &zoneCode, &temperatureClass, &hazmat, &status); err != nil {
			return nil, err
		}
		z, err := rehydrateZone(site, areaCode, zoneCode, temperatureClass, hazmat, status)
		if err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

func rehydrateZone(siteCode, areaCode, zoneCode, temperatureClass string, hazmat bool, status string) (*zone.Zone, error) {
	tc, err := shared.ParseTemperatureClass(temperatureClass)
	if err != nil {
		return nil, err
	}
	st, err := shared.ParseStatus(status)
	if err != nil {
		return nil, err
	}
	return zone.RehydrateZone(siteCode, areaCode, zoneCode, tc, hazmat, st), nil
}
