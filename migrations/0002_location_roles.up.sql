-- ADR-0016: LocationRole on LocationType, and role-conditional DockFlow /
-- Activities on LocationSlot. Every existing row defaults to role
-- 'Storage', preserving current behaviour exactly. Capacity columns on
-- both tables become nullable: a Dock/Yard/WorkCenter/QC/Shipping type or
-- slot may legitimately carry no capacity envelope.

ALTER TABLE location_types
    ADD COLUMN role TEXT NOT NULL DEFAULT 'Storage';

ALTER TABLE location_types
    ALTER COLUMN default_max_weight_kg DROP NOT NULL,
    ALTER COLUMN default_max_volume_m3 DROP NOT NULL;

ALTER TABLE location_types
    DROP CONSTRAINT location_types_default_max_weight_kg_check,
    DROP CONSTRAINT location_types_default_max_volume_m3_check;

ALTER TABLE location_types
    ADD CONSTRAINT location_types_default_max_weight_kg_check
        CHECK (default_max_weight_kg IS NULL OR default_max_weight_kg > 0),
    ADD CONSTRAINT location_types_default_max_volume_m3_check
        CHECK (default_max_volume_m3 IS NULL OR default_max_volume_m3 > 0);

ALTER TABLE location_slots
    ADD COLUMN role TEXT NOT NULL DEFAULT 'Storage',
    ADD COLUMN dock_flow TEXT,
    ADD COLUMN activities TEXT[];

ALTER TABLE location_slots
    ALTER COLUMN max_weight_kg DROP NOT NULL,
    ALTER COLUMN max_volume_m3 DROP NOT NULL;

ALTER TABLE location_slots
    DROP CONSTRAINT location_slots_max_weight_kg_check,
    DROP CONSTRAINT location_slots_max_volume_m3_check;

ALTER TABLE location_slots
    ADD CONSTRAINT location_slots_max_weight_kg_check
        CHECK (max_weight_kg IS NULL OR max_weight_kg > 0),
    ADD CONSTRAINT location_slots_max_volume_m3_check
        CHECK (max_volume_m3 IS NULL OR max_volume_m3 > 0);

CREATE INDEX idx_location_slots_role ON location_slots (site_segment, role);
