package auth

import (
	"context"
	"errors"
	"log"

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

	err := h.repo.UpdateUserProfile(ctx, userID, req)
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

