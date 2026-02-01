package handler

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/manaraph/go-openfga-implementation/pkg/middleware"
	"github.com/openfga/go-sdk/client"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler struct {
	DB      *sqlx.DB
	MongoDB *mongo.Database
	FGA     *client.OpenFgaClient
}

func New(db *sqlx.DB, mongo *mongo.Database, fga *client.OpenFgaClient) *Handler {
	return &Handler{
		DB:      db,
		MongoDB: mongo,
		FGA:     fga,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.health)

	auth := NewAuth(h.DB)
	r.Post("/signup", auth.Signup)
	r.Post("/login", auth.Login)

	authMiddleware := middleware.AuthMiddleware([]byte(os.Getenv("JWT_SECRET")))

	r.Route("/files", func(r chi.Router) {
		r.Use(authMiddleware)
		media := NewFileHandler(h.MongoDB)
		r.Post("/upload", media.Upload)
		r.Get("/", media.GetFiles)
		r.Get("/{id}", media.DownloadFile)
	})
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
