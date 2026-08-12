package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

type ClientHandler struct {
	awgService service.AWGService
}

func NewClientHandler(awgService service.AWGService) *ClientHandler {
	return &ClientHandler{awgService: awgService}
}

func (h *ClientHandler) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.awgService.GetClients()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

func (h *ClientHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	res, err := h.awgService.CreateClient(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *ClientHandler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Client name parameter required"})
		return
	}

	if err := h.awgService.DeleteClient(name); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": fmt.Sprintf("Client '%s' deleted", name)})
}

func (h *ClientHandler) GetClientQRCode(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	qrDataURL, err := h.awgService.GetClientQRCode(name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"name": name, "qr_code_svg": qrDataURL})
}

func (h *ClientHandler) DownloadClientConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	configText, err := h.awgService.GetClientConfigText(name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/x-wireguard-config")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.conf\"", name))
	w.Write([]byte(configText))
}
