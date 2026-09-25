-- ADR-0017: optional physical geometry — position/dimensions/pick-sequence
-- on location_slots, an optional travel centreline on aisles, and a new
-- fixed_structures table for site-scoped obstacles (walls, columns,
-- offices, conveyors). Every column is nullable: geometry is opt-in, and
-- every row registered before this migration keeps drawing exactly as it
-- did (grid layout, no scale).

ALTER TABLE location_slots
    ADD COLUMN x_m           DOUBLE PRECISION,
    ADD COLUMN y_m           DOUBLE PRECISION,
    ADD COLUMN z_m           DOUBLE PRECISION,
    ADD COLUMN width_m       DOUBLE PRECISION,
    ADD COLUMN depth_m       DOUBLE PRECISION,
    ADD COLUMN height_m      DOUBLE PRECISION,
    ADD COLUMN pick_sequence INTEGER;

ALTER TABLE location_slots
    ADD CONSTRAINT location_slots_z_m_check CHECK (z_m IS NULL OR z_m >= 0),
    ADD CONSTRAINT location_slots_width_m_check CHECK (width_m IS NULL OR width_m > 0),
    ADD CONSTRAINT location_slots_depth_m_check CHECK (depth_m IS NULL OR depth_m > 0),
    ADD CONSTRAINT location_slots_height_m_check CHECK (height_m IS NULL OR height_m > 0),
    ADD CONSTRAINT location_slots_pick_sequence_check CHECK (pick_sequence IS NULL OR pick_sequence >= 0);

-- Geometry is all-or-nothing per slot: either every position/dimension
-- column is set, or none are. Guards against a half-written row that
-- SetGeometry's domain invariant (both position and dimensions supplied
-- together) would never itself produce.
ALTER TABLE location_slots
    ADD CONSTRAINT location_slots_geometry_all_or_nothing CHECK (
        (x_m IS NULL AND y_m IS NULL AND z_m IS NULL
            AND width_m IS NULL AND depth_m IS NULL AND height_m IS NULL)
        OR
        (x_m IS NOT NULL AND y_m IS NOT NULL AND z_m IS NOT NULL
            AND width_m IS NOT NULL AND depth_m IS NOT NULL AND height_m IS NOT NULL)
    );

ALTER TABLE aisles
    ADD COLUMN centreline_start_x_m DOUBLE PRECISION,
    ADD COLUMN centreline_start_y_m DOUBLE PRECISION,
    ADD COLUMN centreline_start_z_m DOUBLE PRECISION,
    ADD COLUMN centreline_end_x_m   DOUBLE PRECISION,
    ADD COLUMN centreline_end_y_m   DOUBLE PRECISION,
    ADD COLUMN centreline_end_z_m   DOUBLE PRECISION;

ALTER TABLE aisles
    ADD CONSTRAINT aisles_centreline_all_or_nothing CHECK (
        (centreline_start_x_m IS NULL AND centreline_start_y_m IS NULL AND centreline_start_z_m IS NULL
            AND centreline_end_x_m IS NULL AND centreline_end_y_m IS NULL AND centreline_end_z_m IS NULL)
        OR
        (centreline_start_x_m IS NOT NULL AND centreline_start_y_m IS NOT NULL AND centreline_start_z_m IS NOT NULL
            AND centreline_end_x_m IS NOT NULL AND centreline_end_y_m IS NOT NULL AND centreline_end_z_m IS NOT NULL)
    );

CREATE TABLE fixed_structures (
    id        TEXT PRIMARY KEY,
    site_code TEXT NOT NULL REFERENCES sites (code),
    kind      TEXT NOT NULL,
    x_m       DOUBLE PRECISION NOT NULL,
    y_m       DOUBLE PRECISION NOT NULL,
    z_m       DOUBLE PRECISION NOT NULL CHECK (z_m >= 0),
    width_m   DOUBLE PRECISION NOT NULL CHECK (width_m > 0),
    depth_m   DOUBLE PRECISION NOT NULL CHECK (depth_m > 0),
    height_m  DOUBLE PRECISION NOT NULL CHECK (height_m > 0),
    label     TEXT NOT NULL
);

CREATE INDEX idx_fixed_structures_site_code ON fixed_structures (site_code);
