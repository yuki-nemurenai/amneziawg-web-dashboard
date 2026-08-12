package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

type ServerHandler struct {
	awgService service.AWGService
}

func NewServerHandler(awgService service.AWGService) *ServerHandler {
	return &ServerHandler{awgService: awgService}
}

func (h *ServerHandler) GetServerConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.awgService.GetServerConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (h *ServerHandler) UpdateServerConfig(w http.ResponseWriter, r *http.Request) {
	var cfg domain.ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	if err := h.awgService.UpdateServerConfig(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Server settings updated successfully"})
}
