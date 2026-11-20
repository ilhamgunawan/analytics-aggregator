-- Time the event happened on the client, normalized to UTC.
-- Nullable so rows inserted before this migration stay valid.
ALTER TABLE events
  ADD COLUMN occurred_at TIMESTAMPTZ;
CREATE INDEX idx_events_occurred_at ON events(occurred_at);
