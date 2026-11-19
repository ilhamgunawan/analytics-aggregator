//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"analytics-aggregator/internal/adapter/repository/sqlc"
	"analytics-aggregator/internal/core/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// newTestEvent builds an event that satisfies the events table constraints
// (raw_data is NOT NULL JSONB) with the given ID and occurred_at.
func newTestEvent(id uuid.UUID, occurredAt time.Time) domain.Event {
	return domain.Event{
		ID:         id,
		RawData:    []byte(`{"event_type":"click"}`),
		ClientIP:   "127.0.0.1",
		OccurredAt: occurredAt,
	}
}

func TestPostgresEventRepository_CreateEvents_OccurredAt(t *testing.T) {
	base := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

	// Fixed IDs per case so expectedOccurredAt can be keyed by them.
	singleID := uuid.New()
	batchIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	precisionID := uuid.New()
	nonUTCID := uuid.New()

	subSecond := base.Add(123456789 * time.Nanosecond) // 09:00:00.123456789
	wib := time.FixedZone("WIB", 7*3600)

	tests := []struct {
		name   string
		events []domain.Event
		// expectedOccurredAt is keyed by event ID; each row's occurred_at
		// column must equal this value.
		expectedOccurredAt map[uuid.UUID]time.Time
	}{
		{
			name:   "AC5 single event persists occurred_at",
			events: []domain.Event{newTestEvent(singleID, base)},
			expectedOccurredAt: map[uuid.UUID]time.Time{
				singleID: base,
			},
		},
		{
			name: "AC5 batch with distinct occurred_at persists each row's value",
			events: []domain.Event{
				newTestEvent(batchIDs[0], base),
				newTestEvent(batchIDs[1], base.Add(1*time.Minute)),
				newTestEvent(batchIDs[2], base.Add(2*time.Minute)),
			},
			expectedOccurredAt: map[uuid.UUID]time.Time{
				batchIDs[0]: base,
				batchIDs[1]: base.Add(1 * time.Minute),
				batchIDs[2]: base.Add(2 * time.Minute),
			},
		},
		{
			// Postgres TIMESTAMPTZ stores microseconds, so nanoseconds are dropped.
			name:   "AC5 sub-second precision is preserved to microseconds",
			events: []domain.Event{newTestEvent(precisionID, subSecond)},
			expectedOccurredAt: map[uuid.UUID]time.Time{
				precisionID: subSecond.Truncate(time.Microsecond), // 09:00:00.123456
			},
		},
		{
			// 16:00 WIB (+07:00) is the same instant as 09:00 UTC.
			name:   "AC5 non-UTC input is stored as the same instant",
			events: []domain.Event{newTestEvent(nonUTCID, base.In(wib))},
			expectedOccurredAt: map[uuid.UUID]time.Time{
				nonUTCID: base,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pool := setupPostgres(t)
			repo := &PostgresEventRepository{queries: sqlc.New(pool)}

			n, err := repo.CreateEvents(ctx, tt.events)
			require.NoError(t, err)
			assert.Equal(t, int64(len(tt.events)), n)

			for id, want := range tt.expectedOccurredAt {
				var got time.Time
				err := pool.QueryRow(ctx, `SELECT occurred_at FROM events WHERE id = $1`, id).Scan(&got)
				require.NoError(t, err)
				assert.True(t, want.Equal(got), "id=%s want=%s got=%s", id, want, got)
			}

		})
	}
}

// setupPostgres starts an ephemeral Postgres, applies db/migrations and
// returns a connected pool. Container teardown is registered with t.Cleanup.
//
// TODO: implement with testcontainers-go; signature will become
// func setupPostgres(t *testing.T) *pgxpool.Pool
func setupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	require.NotNil(t, testPool, "testPool not initialised; run with -tags=integration")

	t.Cleanup(func() {
		_, err := testPool.Exec(context.Background(),
			`TRUNCATE events, dead_letter_events`)
		require.NoError(t, err)
	})
	return testPool
}
