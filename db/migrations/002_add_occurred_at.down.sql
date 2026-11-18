DROP INDEX IF EXISTS idx_events_occurred_at;
ALTER TABLE events
  DROP COLUMN IF EXISTS occurred_at;
