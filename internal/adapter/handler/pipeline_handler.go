package handler

import (
	"analytics-aggregator/internal/core/domain"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (h *Handler) ProcessAndStoreEvents(w http.ResponseWriter, r *http.Request) error {
	var processEventReq ProcessEventRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&processEventReq); err != nil {
		return BadRequest(fmt.Sprintf("Invalid request payload : %v", err), err)
	}

	if processEventReq.EventType == "" {
		return BadRequest("event_type is required", errors.New("mandatory field not provided"))
	}

	now := h.now().UTC()

	// Fallback occurred_at to server time when missing. occurred_at invalid value validation is handled by decoder.Decode.
	if processEventReq.OccurredAt == nil {
		processEventReq.OccurredAt = &now
	}

	// Reject if occurred_at is far future time.
	if err := h.isOccurredAtBeyondAllowedClockDrift(processEventReq.OccurredAt, now); err != nil {
		return BadRequest(fmt.Sprintf("Invalid request payload : %v", err), err)
	}

	// Normalize to UTC
	occurredAtUtc := processEventReq.OccurredAt.UTC()

	body := ProcessEventRequest{
		UserID:     processEventReq.UserID,
		EventType:  processEventReq.EventType,
		ClientIP:   processEventReq.ClientIP,
		OccurredAt: &occurredAtUtc,
		Payload:    processEventReq.Payload,
	}

	rawDataBytes, err := json.Marshal(body)
	if err != nil {
		return InternalError(fmt.Sprintf("Failed to marshal raw data : %v", rawDataBytes), err)
	}
	event := &domain.Event{
		ID:         uuid.New(),
		RawData:    rawDataBytes,
		ClientIP:   body.ClientIP,
		OccurredAt: *body.OccurredAt,
	}

	affectedRecs, err := h.service.ProcessAndStore(r.Context(), event)
	if err != nil {
		return InternalError("pipeline ingestion failed", err)
	}

	resp := ProcessEventResponse{
		Status:  "accepted",
		Message: fmt.Sprintf("%d event queued for processing", affectedRecs),
		Id:      event.ID.String(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	return json.NewEncoder(w).Encode(resp)
}

func (h *Handler) isOccurredAtBeyondAllowedClockDrift(occ *time.Time, now time.Time) error {
	if occ.After(now.Add(h.config.AllowedClockDrift)) {
		return errors.New("far future time is not allowed for occurred_at")
	}

	return nil
}
