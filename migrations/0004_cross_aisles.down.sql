-- Reverts 0004_cross_aisles.up.sql.

DROP TABLE IF EXISTS cross_aisles;

ALTER TABLE zones
    DROP CONSTRAINT IF EXISTS zones_bay_pitch_m_check,
    DROP CONSTRAINT IF EXISTS zones_level_pitch_m_check,
    DROP COLUMN IF EXISTS bay_pitch_m,
    DROP COLUMN IF EXISTS level_pitch_m;
