package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

// ServerHandler serves /api/server: the interface settings and obfuscation
// parameters.
type ServerHandler struct {
	awgService service.AWGService
}

// NewServerHandler returns a ServerHandler backed by awgService.
func NewServerHandler(awgService service.AWGService) *ServerHandler {
	return &ServerHandler{awgService: awgService}
}

// GetServerConfig serves GET /api/server.
func (h *ServerHandler) GetServerConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.awgService.GetServerConfig(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// UpdateServerConfig serves POST /api/server. The peers in the body are
// ignored: clients are managed through /api/clients.
func (h *ServerHandler) UpdateServerConfig(w http.ResponseWriter, r *http.Request) {
	var cfg domain.ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	if err := h.awgService.UpdateServerConfig(r.Context(), &cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Server settings updated successfully"})
}
