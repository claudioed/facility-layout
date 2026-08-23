CREATE TABLE sites (
    code   TEXT PRIMARY KEY,
    name   TEXT NOT NULL,
    status TEXT NOT NULL
);

CREATE TABLE zones (
    id                TEXT PRIMARY KEY,
    site_code         TEXT NOT NULL REFERENCES sites (code),
    area_code         TEXT NOT NULL,
    zone_code         TEXT NOT NULL,
    temperature_class TEXT NOT NULL,
    hazmat            BOOLEAN NOT NULL,
    status            TEXT NOT NULL,
    UNIQUE (site_code, area_code, zone_code)
);

CREATE INDEX idx_zones_site_code ON zones (site_code);

CREATE TABLE aisles (
    id            TEXT PRIMARY KEY,
    zone_id       TEXT NOT NULL REFERENCES zones (id),
    aisle_code    TEXT NOT NULL,
    sequence_hint INTEGER NOT NULL CHECK (sequence_hint >= 0),
    direction     TEXT NOT NULL,
    status        TEXT NOT NULL,
    UNIQUE (zone_id, aisle_code)
);

CREATE INDEX idx_aisles_zone_id ON aisles (zone_id);

CREATE TABLE location_types (
    name                  TEXT PRIMARY KEY,
    default_max_weight_kg DOUBLE PRECISION NOT NULL CHECK (default_max_weight_kg > 0),
    default_max_volume_m3 DOUBLE PRECISION NOT NULL CHECK (default_max_volume_m3 > 0)
);

CREATE TABLE placement_rules (
    id                TEXT PRIMARY KEY,
    location_type     TEXT NOT NULL REFERENCES location_types (name),
    effect            TEXT NOT NULL,
    zone_code         TEXT,
    temperature_class TEXT,
    hazmat            BOOLEAN
);

CREATE INDEX idx_placement_rules_location_type ON placement_rules (location_type);

-- The seven LocationCode segments are modelled as real columns, not just a
-- serialized string, so the zone-grid read model can query by
-- aisle/bay/level directly instead of parsing text at read time. `code` is
-- still the primary key: the LocationCode IS the slot's identity.
CREATE TABLE location_slots (
    code             TEXT PRIMARY KEY,
    site_segment     TEXT NOT NULL,
    area_segment     TEXT NOT NULL,
    zone_segment     TEXT NOT NULL,
    aisle_segment    TEXT NOT NULL,
    bay_segment      TEXT NOT NULL,
    level_segment    TEXT NOT NULL,
    position_segment TEXT NOT NULL,
    zone_id          TEXT NOT NULL REFERENCES zones (id),
    aisle_id         TEXT NOT NULL REFERENCES aisles (id),
    location_type    TEXT NOT NULL REFERENCES location_types (name),
    max_weight_kg    DOUBLE PRECISION NOT NULL CHECK (max_weight_kg > 0),
    max_volume_m3    DOUBLE PRECISION NOT NULL CHECK (max_volume_m3 > 0),
    status           TEXT NOT NULL
);

CREATE INDEX idx_location_slots_aisle_id ON location_slots (aisle_id);
CREATE INDEX idx_location_slots_zone_id ON location_slots (zone_id);
CREATE INDEX idx_location_slots_grid ON location_slots (zone_id, aisle_segment, bay_segment, level_segment);

CREATE TABLE events (
    id          BIGSERIAL PRIMARY KEY,
    event_name  TEXT NOT NULL,
    event_type  TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    payload     JSONB NOT NULL
);
