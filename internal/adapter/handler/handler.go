package handler

import (
	"analytics-aggregator/internal/config"
	"analytics-aggregator/internal/core/domain"
	"analytics-aggregator/internal/core/service"
	"analytics-aggregator/internal/ctxkey"
	"context"
	"net/http"
	"time"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

type pipelineService interface {
	ProcessAndStore(ctx context.Context, e *domain.Event) (int64, error)
}

type Handler struct {
	service pipelineService
	config  *config.Config
	now     func() time.Time
}

func NewHandler(s *service.PipelineService, cfg *config.Config) *Handler {
	return &Handler{
		service: s,
		config:  cfg,
		now:     time.Now,
	}
}

func (h *Handler) MakeHandler(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			// centralized error handling
			h.HandleError(w, r, err)
		}
	}
}

// GetIdempotencyKey safely retrieves the key from context
func GetIdempotencyKey(ctx context.Context) string {
	val, ok := ctx.Value(ctxkey.IdempotencyKey).(string)
	if !ok {
		return ""
	}
	return val
}
