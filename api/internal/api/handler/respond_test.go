package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

func TestWriteErrorMapsDomainErrorsToStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: fmt.Errorf("peer %q: %w", "bob", domain.ErrNotFound), want: http.StatusNotFound},
		{name: "invalid input", err: fmt.Errorf("%w: client name is empty", domain.ErrInvalidInput), want: http.StatusBadRequest},
		{name: "conflict", err: fmt.Errorf("%w: peer already exists", domain.ErrConflict), want: http.StatusConflict},
		{name: "unauthorized", err: fmt.Errorf("%w: bad token", domain.ErrUnauthorized), want: http.StatusUnauthorized},
		{name: "other", err: errors.New("connection refused"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, httptest.NewRequest(http.MethodGet, "/api/clients", nil), tt.err)

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestWriteErrorHidesServerFaults(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil), errors.New(`relation "peers" does not exist`))

	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error != "Internal Server Error" {
		t.Errorf("error = %q, want %q", body.Error, "Internal Server Error")
	}
}
