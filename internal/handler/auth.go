package handler

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/gommon/log"
	"github.com/manaraph/go-openfga-implementation/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct{}

func NewAuth() *AuthHandler {
	return &AuthHandler{}
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// POST /signup
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	log.Infof("request: %v", r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "error hashing password: " + err.Error(),
		})
		return
	}

	_, err = db.DB.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", req.Username, string(hashed))
	if err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{
			"message": "user creation failed: " + err.Error(),
		})
		return
	}

	apiResponse(w, http.StatusCreated, map[string]string{
		"message": "user created",
	})
}
