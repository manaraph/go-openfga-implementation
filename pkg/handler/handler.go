package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler struct {
	DB      *sqlx.DB
	MongoDB *mongo.Database
}

func New(db *sqlx.DB, mongo *mongo.Database) *Handler {
	return &Handler{
		DB:      db,
		MongoDB: mongo,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.health)

	auth := NewAuth(h.DB)
	r.Post("/signup", auth.Signup)
	r.Post("/login", auth.Login)

	media := NewFileHandler(h.MongoDB)
	r.Post("/upload", media.Upload)
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
