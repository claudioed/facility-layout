-- Reverts 0002_location_roles.up.sql.

DROP INDEX IF EXISTS idx_location_slots_role;

ALTER TABLE location_slots
    DROP CONSTRAINT IF EXISTS location_slots_max_weight_kg_check,
    DROP CONSTRAINT IF EXISTS location_slots_max_volume_m3_check;

ALTER TABLE location_slots
    ADD CONSTRAINT location_slots_max_weight_kg_check CHECK (max_weight_kg > 0),
    ADD CONSTRAINT location_slots_max_volume_m3_check CHECK (max_volume_m3 > 0);

ALTER TABLE location_slots
    ALTER COLUMN max_weight_kg SET NOT NULL,
    ALTER COLUMN max_volume_m3 SET NOT NULL;

ALTER TABLE location_slots
    DROP COLUMN activities,
    DROP COLUMN dock_flow,
    DROP COLUMN role;

ALTER TABLE location_types
    DROP CONSTRAINT IF EXISTS location_types_default_max_weight_kg_check,
    DROP CONSTRAINT IF EXISTS location_types_default_max_volume_m3_check;

ALTER TABLE location_types
    ADD CONSTRAINT location_types_default_max_weight_kg_check CHECK (default_max_weight_kg > 0),
    ADD CONSTRAINT location_types_default_max_volume_m3_check CHECK (default_max_volume_m3 > 0);

ALTER TABLE location_types
    ALTER COLUMN default_max_weight_kg SET NOT NULL,
    ALTER COLUMN default_max_volume_m3 SET NOT NULL;

ALTER TABLE location_types
    DROP COLUMN role;
