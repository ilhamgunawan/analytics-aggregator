package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"analytics-aggregator/internal/config"
	"analytics-aggregator/internal/core/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Spec: docs/capture-client-timestamp.md (Technical AC 1-4)
//
// Requires these seams in handler.go:
//   - eventProcessor interface, used as the type of Handler.service
//   - Handler.now func() time.Time (injectable clock)

// fakeEventProcessor is a handwritten fake of eventProcessor. It records the
// last event it received so tests can assert on it.
type fakePipelineService struct {
	calls     int
	lastEvent *domain.Event
	err       error
}

func (f *fakePipelineService) ProcessAndStore(ctx context.Context, e *domain.Event) (int64, error) {
	f.calls++
	f.lastEvent = e
	if f.err != nil {
		return 0, f.err
	}
	return 1, nil
}

func TestHandler_ProcessAndStoreEvents_OccurredAt(t *testing.T) {
	// Fixed server clock and allowed drift shared by all cases.
	fixedNow := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	const allowedDrift = 5 * time.Minute

	tests := []struct {
		name string
		// body is the raw request JSON, so each case controls exactly how
		// occurred_at is encoded (absent, valid or invalid string, ...).
		body               string
		expectedStatus     int
		expectServiceCall  bool
		expectedOccurredAt time.Time // checked only when expectServiceCall is true
		expectedErrMsg     string    // substring of the 400 body; empty = not checked
	}{
		// ---- Technical AC 1: Happy path (Product AC 1) ----
		{
			name:               "positive",
			body:               `{"event_type":"click","occurred_at":"2026-01-15T09:59:00Z"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: time.Date(2026, 1, 15, 9, 59, 0, 0, time.UTC),
		},
		{
			name:               "positive_timezone",
			body:               `{"event_type":"click","occurred_at":"2026-01-15T16:59:00+07:00"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: time.Date(2026, 1, 15, 9, 59, 0, 0, time.UTC),
		},
		{
			name:               "positive_offset",
			body:               `{"event_type":"click","occurred_at":"2026-01-15T10:04:59Z"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: time.Date(2026, 1, 15, 10, 4, 59, 0, time.UTC),
		},

		// ---- Technical AC 2: Missing field (Product AC 3) ----
		{
			name:               "positive_empty_fallback",
			body:               `{"event_type":"click"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: fixedNow,
		},

		// ---- Technical AC 3: Invalid format (Product AC 4) ----
		// TODO: set expectedErrMsg to "occurred_at" once the handler returns a
		// field-specific message for parse errors.
		{
			name:           "negative_invalid",
			body:           `{"event_type":"click","occurred_at":"01-01-2026"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative_non_date",
			body:           `{"event_type":"click","occurred_at":"invalid-date"}`,
			expectedStatus: http.StatusBadRequest,
		},

		// ---- Technical AC 4: Future time (Product AC 4) ----
		{
			name:           "negative_beyond_offset",
			body:           `{"event_type":"click","occurred_at":"2026-01-15T10:05:01Z"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "occurred_at",
		},
		{
			name:           "negative_far_beyond_offset",
			body:           `{"event_type":"click","occurred_at":"9999-01-01T00:00:00Z"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "occurred_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 1. Arrange
			fp := &fakePipelineService{}
			h := &Handler{
				service: fp,
				config:  &config.Config{AllowedClockDrift: allowedDrift},
				now:     func() time.Time { return fixedNow },
			}
			req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			// 2. Act
			h.MakeHandler(h.ProcessAndStoreEvents)(rec, req)

			// 3. Assert
			require.Equal(t, tt.expectedStatus, rec.Code, "body: %s", rec.Body.String())

			if tt.expectServiceCall {
				require.Equal(t, 1, fp.calls, "service should be called exactly once")
				got := fp.lastEvent.OccurredAt
				assert.True(t, got.Equal(tt.expectedOccurredAt),
					"OccurredAt = %s, want %s", got, tt.expectedOccurredAt)
				assert.Equal(t, time.UTC, got.Location())
			} else {
				assert.Equal(t, 0, fp.calls, "service must not be called on rejection")
				if tt.expectedErrMsg != "" {
					assert.Contains(t, rec.Body.String(), tt.expectedErrMsg)
				}
			}
		})
	}
}
