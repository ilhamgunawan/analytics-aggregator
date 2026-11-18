package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"analytics-aggregator/internal/core/domain"

	"github.com/stretchr/testify/mock"
)

// Spec: docs/capture-client-timestamp.md (Technical AC 1-4)
//
// Seams the implementation must introduce before these tests can be filled in:
//   1. Handler.service must depend on an interface defined here (consumer side),
//      not on *service.PipelineService, so the mock below can be injected.
//      eventProcessor is the proposed shape.
//   2. Handler needs an injectable clock (e.g. `now func() time.Time`) so the
//      "missing field" and "future time" cases are deterministic.
//   3. The allowed clock drift must be configurable (e.g. config.Config field)
//      so the boundary can be tested without sleeping.
//   4. ProcessEventRequest must tell "absent" apart from explicit `null`
//      (AC 3 rejects null; AC 2 accepts absent). A plain *time.Time can't do
//      that — use json.RawMessage or a custom type.

// eventProcessor is the dependency ProcessAndStoreEvents consumes.
type eventProcessor interface {
	ProcessAndStore(ctx context.Context, e *domain.Event) (int64, error)
}

// MockEventProcessor is a testify mock of eventProcessor.
type MockEventProcessor struct {
	mock.Mock
}

func (m *MockEventProcessor) ProcessAndStore(ctx context.Context, e *domain.Event) (int64, error) {
	args := m.Called(ctx, e)
	return args.Get(0).(int64), args.Error(1)
}

var _ eventProcessor = (*MockEventProcessor)(nil)

func TestHandler_ProcessAndStoreEvents_OccurredAt(t *testing.T) {
	// Fixed server clock and allowed drift shared by all cases.
	fixedNow := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	const allowedDrift = 5 * time.Minute

	tests := []struct {
		name string
		// body is the raw request JSON, so each case controls exactly how
		// occurred_at is encoded (absent, null, number, string, ...).
		body               string
		expectedStatus     int
		expectServiceCall  bool
		expectedOccurredAt time.Time // checked only when expectServiceCall is true
	}{
		// ---- Technical AC 1: Happy path (Product AC 1) ----
		{
			name:               "AC1 valid RFC3339 UTC is accepted and passed through",
			body:               `{"event_type":"click","occurred_at":"2026-01-15T09:59:00Z"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: time.Date(2026, 1, 15, 9, 59, 0, 0, time.UTC),
		},
		{
			name:               "AC1 valid timestamp with offset is normalised to UTC",
			body:               `{"event_type":"click","occurred_at":"2026-01-15T16:59:00+07:00"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: time.Date(2026, 1, 15, 9, 59, 0, 0, time.UTC),
		},
		{
			name:               "AC1 future time within allowed drift is accepted",
			body:               `{"event_type":"click","occurred_at":"2026-01-15T10:04:59Z"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: time.Date(2026, 1, 15, 10, 4, 59, 0, time.UTC),
		},

		// ---- Technical AC 2: Missing field (Product AC 3) ----
		{
			name:               "AC2 missing occurred_at falls back to server time",
			body:               `{"event_type":"click"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: fixedNow,
		},

		// ---- Technical AC 3: Invalid format (Product AC 4) ----
		{
			name:           "AC3 non-RFC3339 date string is rejected",
			body:           `{"event_type":"click","occurred_at":"01-01-2026"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "AC3 empty string is rejected",
			body:           `{"event_type":"click","occurred_at":""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "AC3 numeric zero is rejected",
			body:           `{"event_type":"click","occurred_at":0}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "AC3 explicit null is rejected",
			body:           `{"event_type":"click","occurred_at":null}`,
			expectedStatus: http.StatusBadRequest,
		},

		// ---- Technical AC 4: Future time (Product AC 4) ----
		{
			name:           "AC4 future time just beyond allowed drift is rejected",
			body:           `{"event_type":"click","occurred_at":"2026-01-15T10:05:01Z"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "AC4 far future time is rejected",
			body:           `{"event_type":"click","occurred_at":"9999-01-01T00:00:00Z"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Skip("TODO: implement once the handler accepts eventProcessor and an injectable clock")

			// 1. Arrange
			mp := new(MockEventProcessor)
			if tt.expectServiceCall {
				mp.On("ProcessAndStore", mock.Anything, mock.MatchedBy(func(e *domain.Event) bool {
					// TODO: compare e.OccurredAt once the field exists:
					//   return e.OccurredAt.Equal(tt.expectedOccurredAt) &&
					//          e.OccurredAt.Location() == time.UTC
					return true
				})).Return(int64(1), nil).Once()
			}
			// TODO: build the handler with the mock, the fixed clock and the drift, e.g.
			//   h := &Handler{service: mp, now: func() time.Time { return fixedNow }, config: &config.Config{AllowedClockDrift: allowedDrift}}
			//   req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(tt.body))
			//   rec := httptest.NewRecorder()
			_, _ = fixedNow, allowedDrift

			// 2. Act
			// TODO: h.MakeHandler(h.ProcessAndStoreEvents)(rec, req)

			// 3. Assert
			// TODO: assert.Equal(t, tt.expectedStatus, rec.Code)
			// TODO: on 400, assert the body names occurred_at (the "clear error message" in Product AC 4)
			if !tt.expectServiceCall {
				mp.AssertNotCalled(t, "ProcessAndStore", mock.Anything, mock.Anything)
			}
			mp.AssertExpectations(t)
		})
	}
}
