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

// RegisterHandler godoc
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param   request body RegisterRequest true "Register user request"
// @Success 201 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /auth/register-with-email [post]
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

// LoginHandler godoc
// @Summary Login a user
// @Description Authenticate a user and return a JWT token
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param   request body LoginRequest true "Login user request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /auth/login [post]
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

// CompleteProfileHandler godoc
// @Summary Complete user profile
// @Description Complete user's registration
// @Tags Auth
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param   request body CompleteProfileRequest true "Complete profile request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/complete-profile [post]
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

// Verify TokenHandler godoc
// @Summary Verify JWT token
// @Description Check if the provided JWT token is valid
// @Tags Auth
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/verify-token [get]
func (h *AuthHandler) VerifyTokenHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not found")
		return
	}

	user, err := h.service.VerifyToken(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Invalid token", err.Error())
		return
	}

	response.Success(w, user, "Token is valid")
}

// ForgetPasswordHandler godoc
// @Summary Initiate password reset
// @Description Send OTP to user's email for password reset
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param   request body ForgetPasswordRequest true "Forget password request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /auth/forget-password [post]
func (h *AuthHandler) ForgetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req ForgetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", err.Error())
		return
	}

	if req.Email == "" {
		response.Error(w, http.StatusBadRequest, "Missing required fields", map[string]string{
			"email": "Email is required",
		})
		return
	}

	success, err := h.service.ForgetPassword(r.Context(), req.Email)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to process request", err.Error())
		return
	}

	response.Success(w, success, "OTP sent to email successfully")

}

// ResetPasswordHandler godoc
// @Summary Reset user password
// @Description Reset password using OTP sent to email
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param   request body ResetPasswordRequest true "Reset password request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", err.Error())
		return
	}

	if req.NewPassword == "" || req.OTP == "" || req.Email == "" {
		response.Error(w, http.StatusBadRequest, "Missing required fields", map[string]string{
			"new_password": "New Password is required",
			"otp":          "OTP is required",
			"email":        "Email is required",
		})
		return
	}

	success, err := h.service.ResetPassword(r.Context(), req.Email, req.OTP, req.NewPassword)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to reset password", err.Error())
		return
	}

	response.Success(w, success, "Password reset successfully")
}

// UpdateUserProfileHandler godoc
// @Summary Update user profile
// @Description Update user's profile information
// @Tags Auth
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param request body UpdateUserProfileRequest true "Update profile request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/update-profile [patch]
func (h *AuthHandler) UpdateUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid input", err.Error())
		return
	}

	userID, ok := GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not found")
		return
	}

	err := h.service.UpdateUserProfile(r.Context(), userID, req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update user profile", err.Error())
		return
	}

	response.Success(w, "OK", "User Profile Updated Successfully")

}

// GetUserDetailsHandler godoc
// @Summary Get user details
// @Description Retrieve detailed information about the authenticated user
// @Tags Auth
// @Produce  json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) GetUserDetailsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not found")
		return
	}

	userDetails, err := h.service.GetUserDetails(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch user details", err.Error())
		return
	}

	response.Success(w, userDetails, "User Profile Retrieved Successfully")

}
