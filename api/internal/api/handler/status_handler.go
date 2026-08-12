package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

type StatusHandler struct {
	awgService service.AWGService
}

func NewStatusHandler(awgService service.AWGService) *StatusHandler {
	return &StatusHandler{awgService: awgService}
}

func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.awgService.GetSystemStatus()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
