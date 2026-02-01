package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB *sqlx.DB
}

func NewAuth(db *sqlx.DB) *AuthHandler {
	return &AuthHandler{
		DB: db,
	}
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

	_, err = h.DB.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", req.Username, string(hashed))
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

// POST /login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	var userId int
	var hashed string
	err := h.DB.QueryRow("SELECT id, password FROM users WHERE username=$1", req.Username).Scan(&userId, &hashed)
	if err != nil {
		apiResponse(w, http.StatusUnauthorized, map[string]string{
			"message": "invalid credentials: " + err.Error(),
		})
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(req.Password)); err != nil {
		apiResponse(w, http.StatusUnauthorized, map[string]string{
			"message": "invalid credentials: " + err.Error(),
		})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userId,
		"username": req.Username,
		"exp":      time.Now().Add(time.Hour * 3).Unix(),
	})

	secret := os.Getenv("JWT_SECRET")
	t, err := token.SignedString([]byte(secret))
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "error signing token: " + err.Error(),
		})
		return
	}

	apiResponse(w, http.StatusOK, map[string]string{
		"message": "success",
		"token":   t,
	})
}
