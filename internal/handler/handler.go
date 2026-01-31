package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.health)

	auth := NewAuth()
	r.Post("/signup", auth.Signup)
	r.Post("/login", auth.Login)
}

func apiResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	apiResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "API Working",
	})
}
