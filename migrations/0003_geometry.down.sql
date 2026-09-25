-- Reverts 0003_geometry.up.sql.

DROP TABLE IF EXISTS fixed_structures;

ALTER TABLE aisles
    DROP CONSTRAINT IF EXISTS aisles_centreline_all_or_nothing,
    DROP COLUMN IF EXISTS centreline_start_x_m,
    DROP COLUMN IF EXISTS centreline_start_y_m,
    DROP COLUMN IF EXISTS centreline_start_z_m,
    DROP COLUMN IF EXISTS centreline_end_x_m,
    DROP COLUMN IF EXISTS centreline_end_y_m,
    DROP COLUMN IF EXISTS centreline_end_z_m;

ALTER TABLE location_slots
    DROP CONSTRAINT IF EXISTS location_slots_geometry_all_or_nothing,
    DROP CONSTRAINT IF EXISTS location_slots_z_m_check,
    DROP CONSTRAINT IF EXISTS location_slots_width_m_check,
    DROP CONSTRAINT IF EXISTS location_slots_depth_m_check,
    DROP CONSTRAINT IF EXISTS location_slots_height_m_check,
    DROP CONSTRAINT IF EXISTS location_slots_pick_sequence_check,
    DROP COLUMN IF EXISTS x_m,
    DROP COLUMN IF EXISTS y_m,
    DROP COLUMN IF EXISTS z_m,
    DROP COLUMN IF EXISTS width_m,
    DROP COLUMN IF EXISTS depth_m,
    DROP COLUMN IF EXISTS height_m,
    DROP COLUMN IF EXISTS pick_sequence;
