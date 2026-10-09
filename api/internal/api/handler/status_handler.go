// Package handler turns HTTP requests into service calls and service results
// into JSON responses and status codes.
package handler

import (
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
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}
