//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"analytics-aggregator/internal/core/domain"

	"github.com/google/uuid"
)

// Spec: docs/capture-client-timestamp.md (Technical AC 5)
//
// Run with: go test -tags=integration ./internal/adapter/repository/...
//
// Prerequisites before this can be filled in:
//   1. Migration adding `occurred_at TIMESTAMPTZ NOT NULL` to events.
//   2. db/queries/events.sql BulkInsertEvents includes occurred_at; regenerate sqlc.
//   3. domain.Event gets an `OccurredAt time.Time` field, mapped in CreateEvents.
//   4. Per testing.MD, use Testcontainers (testcontainers-go postgres module)
//      rather than a mocked DB; setupPostgres below is the place for it.

func TestPostgresEventRepository_CreateEvents_OccurredAt(t *testing.T) {
	base := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		events []domain.Event
		// expectedOccurredAt is keyed by event ID; each row's occurred_at
		// column must equal this value.
		expectedOccurredAt map[uuid.UUID]time.Time
	}{
		{
			name: "AC5 single event persists occurred_at",
			// TODO: set OccurredAt: base once the field exists.
		},
		{
			name: "AC5 batch with distinct occurred_at persists each row's value",
			// TODO: 3 events with base, base+1m, base+2m.
		},
		{
			name: "AC5 sub-second precision is preserved to microseconds",
			// Postgres TIMESTAMPTZ stores microseconds; compare after Truncate(time.Microsecond).
			// TODO: OccurredAt: base.Add(123456789 * time.Nanosecond)
		},
		{
			name: "AC5 non-UTC input is stored as the same instant",
			// TODO: OccurredAt: base.In(time.FixedZone("WIB", 7*3600)); assert with time.Time.Equal.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("TODO: implement once occurred_at column, sqlc query and domain field exist")

			// 1. Arrange
			ctx := context.Background()
			_ = ctx
			_ = base
			// TODO: pool := setupPostgres(t)  // container + migrations, cleaned up via t.Cleanup
			// TODO: repo := &PostgresEventRepository{queries: sqlc.New(pool)}

			// 2. Act
			// TODO: n, err := repo.CreateEvents(ctx, tt.events)
			// TODO: require.NoError(t, err)
			// TODO: assert.Equal(t, int64(len(tt.events)), n)

			// 3. Assert
			// TODO: for id, want := range tt.expectedOccurredAt {
			//     var got time.Time
			//     err := pool.QueryRow(ctx, `SELECT occurred_at FROM events WHERE id = $1`, id).Scan(&got)
			//     require.NoError(t, err)
			//     assert.True(t, want.Truncate(time.Microsecond).Equal(got), "id=%s want=%s got=%s", id, want, got)
			// }
		})
	}
}

// setupPostgres starts an ephemeral Postgres, applies db/migrations and
// returns a connected pool. Container teardown is registered with t.Cleanup.
//
// TODO: implement with testcontainers-go; signature will become
// func setupPostgres(t *testing.T) *pgxpool.Pool
func setupPostgres(t *testing.T) {
	t.Helper()
}
