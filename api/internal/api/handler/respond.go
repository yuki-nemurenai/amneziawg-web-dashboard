package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

// errorResponse is the body of every failed request; the UI shows Error to
// the user.
type errorResponse struct {
	Error string `json:"error"`
}

// messageResponse is the body of a successful request that returns no data.
type messageResponse struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("Failed to write response", "error", err)
	}
}

// writeError responds with the status code of a domain error. Any other error
// is a server fault: it is logged and replaced by a generic message, because it
// can carry SQL or file system details.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := statusOf(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		slog.Error("Request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		msg = http.StatusText(status)
	}
	writeJSON(w, status, errorResponse{Error: msg})
}

func statusOf(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// decodeJSON reads the request body into v and answers 400 if it is not valid
// JSON. It reports whether the handler can go on.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Invalid request payload"})
		return false
	}
	return true
}
