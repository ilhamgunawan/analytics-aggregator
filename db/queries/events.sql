-- name: BulkInsertEvents :copyfrom
INSERT INTO events (id, raw_data, enriched_data, occurred_at)
VALUES ($1, $2, $3, $4);
