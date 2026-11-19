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
	fixedNow := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	const allowedDrift = 5 * time.Minute

	tests := []struct {
		name               string
		body               string
		expectedStatus     int
		expectServiceCall  bool
		expectedOccurredAt time.Time
		expectedErrMsg     string
	}{
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
		{
			name:               "positive_empty_fallback",
			body:               `{"event_type":"click"}`,
			expectedStatus:     http.StatusAccepted,
			expectServiceCall:  true,
			expectedOccurredAt: fixedNow,
		},
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

			fp := &fakePipelineService{}
			h := &Handler{
				service: fp,
				config:  &config.Config{AllowedClockDrift: allowedDrift},
				now:     func() time.Time { return fixedNow },
			}
			req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			h.MakeHandler(h.ProcessAndStoreEvents)(rec, req)

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
