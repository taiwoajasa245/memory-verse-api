package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/taiwoajasa245/memory-verse-api/internal/mail"
	"github.com/taiwoajasa245/memory-verse-api/pkg/util"
)

type AuthService struct {
	repo Repository
	mail *mail.Mailer
}

func NewAuthService(repo Repository, mail *mail.Mailer) AuthService {
	return AuthService{
		repo: repo,
		mail: mail,
	}
}

func (h *AuthService) Register(ctx context.Context, email, password string) (*User, error) {
	if email == "" || password == "" {
		return &User{}, errors.New("invalid email and password")
	}

	hashed, err := util.HashPasswordBcrypt(password)
	if err != nil {
		return &User{}, err
	}

	user := User{Email: email, Password: hashed}

	_, err = h.repo.CreateUser(ctx, user)
	if err != nil {
		log.Printf("Service err: %v", err.Error())
		return &User{}, err
	}

	logInUser, err := h.Login(ctx, email, password)
	if err != nil {
		return &User{}, err
	}

	data := map[string]interface{}{
		"Name":         user.Email,
		"DashboardURL": "https://memoryverse.app/dashboard",
	}

	// Send welcome mail asynchronously
	go func() {
		if err := h.mail.SendHTML(email, "🎉 Welcome to Memory Verse", "welcome.html", data); err != nil {
			log.Printf("failed to send welcome email: %v", err)
		} else {
			log.Println("Email sent successfully")
		}
	}()

	return logInUser, nil
}

func (h *AuthService) Login(ctx context.Context, email, password string) (*User, error) {
	if email == "" || password == "" {
		return &User{}, ErrInvalidCredentials
	}

	user, err := h.repo.GetUserByEmail(ctx, email)
	if err != nil {
		log.Printf("Service err: %v", err.Error())
		return nil, ErrInvalidCredentials
	}

	err = util.ComparePasswordBcrypt(user.Password, password)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := util.GenerateJWT(user.ID, user.Email)
	if err != nil {
		return &User{}, err
	}

	user.Token = token

	return user, nil

}

func (h *AuthService) CompleteUserProfile(ctx context.Context, userID int, req CompleteProfileRequest) error {

	if req.VersePace == "" ||
		req.BibleTranslation == "" ||
		len(req.Inspirations) == 0 ||
		req.UserName == "" ||
		req.SelectedTime.IsZero() {
		return errors.New("incomplete profile data")
	}

	err := h.repo.CompleteUserProfile(ctx, userID, req)
	if err != nil {
		return err
	}

	err = h.repo.UpdateUserInspirations(ctx, userID, req.Inspirations)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	err = h.repo.MarkProfileCompleted(ctx, userID)
	if err != nil {
		return err
	}

	return nil
}

// VerifyToken verifies the JWT token and returns the user if valid.
func (h *AuthService) VerifyToken(ctx context.Context, userId int) (*User, error) {
	user, _, err := h.repo.GetUserWithProfile(ctx, userId)
	if err != nil {
		log.Printf("error fetching user: %v", err)
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (h *AuthService) ForgetPassword(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, ErrInvalidCredentials
	}

	user, err := h.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return false, ErrInvalidCredentials
	}

	// generate OTP
	// 10 minutes expiration
	otp := util.GenerateOTP()
	expiration := time.Now().Add(10 * time.Minute)

	err = h.repo.SavePasswordReset(ctx, email, otp, expiration)
	if err != nil {
		log.Printf("Service err: %v", err.Error())
		return false, ErrInternalServer
	}

	data := map[string]interface{}{
		"Name": user.Email,
		"OTP":  otp,
	}

	go func() {
		h.mail.SendHTML(email, "Reset Your Password OTP", "reset_otp.html", data)
	}()

	return true, nil
}

func (h *AuthService) VerifyOTP(ctx context.Context, email, otp string) (bool, error) {
	savedOTP, expiresAt, err := h.repo.GetPasswordReset(ctx, email)
	if err != nil {
		return false, errors.New("OTP not found")
	}

	if time.Now().After(expiresAt) {
		return false, errors.New("OTP expired")
	}

	if otp != savedOTP {
		return false, errors.New("invalid OTP")
	}

	return true, nil
}

func (h *AuthService) ResetPassword(ctx context.Context, email, otp, newPassword string) (bool, error) {

	ok, err := h.VerifyOTP(ctx, email, otp)
	if !ok || err != nil {
		return false, errors.New("invalid or expired OTP")
	}

	hashed, err := util.HashPasswordBcrypt(newPassword)
	if err != nil {
		return false, err
	}

	// update password
	err = h.repo.UpdateUserPassword(ctx, email, hashed)
	if err != nil {
		return false, err
	}

	// delete OTP in DB
	if err = h.repo.DeletePasswordReset(ctx, email); err != nil {
		log.Printf("failed to delete used OTP: %v", err)
		return false, err
	}

	return true, nil
}



func (h *AuthService) UpdateUserProfile(ctx context.Context, userID int, req UpdateUserProfileRequest) error {

	if req.VersePace == "" ||
		req.BibleTranslation == "" ||
		req.Email == "" ||
		len(req.Inspirations) == 0 ||
		req.UserName == "" ||
		req.SelectedTime.IsZero() {
		return errors.New("incomplete profile data")
	}

	err := h.repo.UpdateUserProfile(ctx, userID, req)
	if err != nil {
		return err
	}

	err = h.repo.UpdateUserInspirations(ctx, userID, req.Inspirations)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}


func (h *AuthService) GetUserDetails(ctx context.Context, userID int) ( *UserDetails, error) {

	UserDetails ,err := h.repo.GetUserDetails(ctx, userID)
	if err != nil {
		return  nil, err
	}

	return UserDetails, nil
}



