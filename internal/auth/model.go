// User model definition
package auth

import "time"

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ForgetPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email"`
	OTP         string `json:"otp"`
	NewPassword string `json:"new_password"`
}

type CompleteProfileRequest struct {
	VersePace           string    `json:"verse_pace"`
	BibleTranslation    string    `json:"bible_translation"`
	EnableNotification  bool      `json:"enable_notification"`
	Inspirations        []string  `json:"inspiration"`
	IsEmailNotification bool      `json:"is_email_notification"`
	IsWebNotification   bool      `json:"is_web_notification"`
	SelectedTime        time.Time `json:"selected_time"`
	UserName            string    `json:"user_name"`
}

type UpdateUserProfileRequest struct {
	VersePace           string    `json:"verse_pace"`
	BibleTranslation    string    `json:"bible_translation"`
	EnableNotification  bool      `json:"enable_notification"`
	Inspirations        []string  `json:"inspiration"`
	IsEmailNotification bool      `json:"is_email_notification"`
	IsWebNotification   bool      `json:"is_web_notification"`
	SelectedTime        time.Time `json:"selected_time"`
	UserName            string    `json:"user_name"`
	Email               string    `json:"email"`
}

type User struct {
	ID                 int        `json:"id"`
	UserName           string     `json:"user_name,omitempty"`
	Email              string     `json:"email"`
	Password           string     `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	Token              string     `json:"token,omitempty"`
	IsProfileCompleted bool       `json:"is_profile_completed,omitempty"`
	VersePace          string     `json:"verse_pace,omitempty"`
	LastVerseSentAt    *time.Time `json:"last_verse_sent_at,omitempty"`
	IsSubscribed       bool       `json:"is_subscribed"`
}

type UserDetails struct {
	ID                 int        `json:"id"`
	UserName           string     `json:"user_name,omitempty"`
	Email              string     `json:"email"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	IsProfileCompleted bool       `json:"is_profile_completed,omitempty"`
	VersePace          string     `json:"verse_pace,omitempty"`
	LastVerseSentAt    *time.Time `json:"last_verse_sent_at,omitempty"`
	IsSubscribed       bool       `json:"is_subscribed"`
	BibleTranslation   string     `json:"bible_translation,omitempty"`
	EnableNotification bool       `json:"enable_notification"`
	Inspirations       []string   `json:"inspirations,omitempty"`
	IsEmailNotification bool      `json:"is_email_notification"`
	IsWebNotification   bool      `json:"is_web_notification"`
	SelectedTime        time.Time `json:"selected_time,omitempty"`
}
