package auth

import (
	"encoding/json"
	"net/http"

	"github.com/taiwoajasa245/memory-verse-api/pkg/response"
)

type AuthHandler struct {
	service AuthService
}

func NewHandler(service AuthService) AuthHandler {
	return AuthHandler{service: service}
}

func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "Missing required fields", map[string]string{
			"email":    "Email is required",
			"password": "Password is required",
		})
		return
	}

	user := User{Email: req.Email, Password: req.Password}

	usr, err := h.service.Register(r.Context(), user.Email, user.Password)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create user", err.Error())
		return
	}

	response.Success(w, usr, "User registered successfully")
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "Missing required fields", map[string]string{
			"email":    "Email is required",
			"password": "Password is required",
		})

		return
	}

	user := &User{Email: req.Email, Password: req.Password}

	user, err := h.service.Login(r.Context(), user.Email, user.Password)
	if err != nil {
		response.Error(w, http.StatusNotFound, "User not found", err.Error())
		return
	}

	response.Success(w, &user, "Ok")
}

func (h *AuthHandler) CompleteProfileHandler(w http.ResponseWriter, r *http.Request) {
	var req CompleteProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid input", err.Error())
		return
	}

	userID, ok := GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not found")
		return
	}

	err := h.service.CompleteUserProfile(r.Context(), userID, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), err.Error())
		return
	}

	response.Success(w, "Profile completed successfully", "OK")
}
