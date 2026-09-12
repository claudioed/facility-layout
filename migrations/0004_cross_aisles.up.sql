-- ADR-0017: cross_aisles (zone-scoped connections between two aisles at a
-- bay ordinal, the travel graph's cross-aisle edges) and optional
-- bay_pitch_m/level_pitch_m overrides on zones (the estimated-distance
-- fallback pitch used when an aisle carries no centreline geometry).

ALTER TABLE zones
    ADD COLUMN bay_pitch_m   DOUBLE PRECISION,
    ADD COLUMN level_pitch_m DOUBLE PRECISION;

ALTER TABLE zones
    ADD CONSTRAINT zones_bay_pitch_m_check CHECK (bay_pitch_m IS NULL OR bay_pitch_m > 0),
    ADD CONSTRAINT zones_level_pitch_m_check CHECK (level_pitch_m IS NULL OR level_pitch_m > 0);

CREATE TABLE cross_aisles (
    zone_id        TEXT NOT NULL REFERENCES zones (id),
    from_aisle     TEXT NOT NULL,
    to_aisle       TEXT NOT NULL,
    at_bay         TEXT NOT NULL,
    decommissioned BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (zone_id, from_aisle, to_aisle, at_bay),
    CHECK (from_aisle <> to_aisle)
);

CREATE INDEX idx_cross_aisles_zone_id ON cross_aisles (zone_id);
