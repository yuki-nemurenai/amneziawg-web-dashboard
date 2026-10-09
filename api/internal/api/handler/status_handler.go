// Package handler turns HTTP requests into service calls and service results
// into JSON responses and status codes.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

// StatusHandler serves /api/status, the live dashboard data.
type StatusHandler struct {
	awgService service.AWGService
}

// NewStatusHandler returns a StatusHandler backed by awgService.
func NewStatusHandler(awgService service.AWGService) *StatusHandler {
	return &StatusHandler{awgService: awgService}
}

// GetStatus serves GET /api/status.
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.awgService.GetSystemStatus(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
