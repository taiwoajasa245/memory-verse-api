package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/taiwoajasa245/memory-verse-api/internal/database"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

// Repository defines the methods the Auth module provides for DB operations.
type Repository interface {
	CreateUser(ctx context.Context, user User) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUserProfile(ctx context.Context, userID int, req CompleteProfileRequest) error
	MarkProfileCompleted(ctx context.Context, userID int) error
	UpdateUserInspirations(ctx context.Context, userID int, inspirations []string) error
	GetUserWithProfile(ctx context.Context, userID int) (*User, *CompleteProfileRequest, error)
	GetAllUsers(ctx context.Context) ([]User, error)
	GetAllUsersWithVersePace(ctx context.Context) ([]User, error)
	UpdateLastVerseSentAt(ctx context.Context, userID int, t time.Time) error
	UnsubscribeUser(ctx context.Context, userID int) error
}

// repository implements Repository.
type repository struct {
	db *sql.DB
}

// NewRepository creates a new auth repository using the shared DB service.
func NewRepository(dbService database.Service) Repository {
	return &repository{db: dbService.DB()}
}

func (r *repository) GetAllUsers(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *repository) CreateUser(ctx context.Context, user User) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Check if email exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.db.QueryRowContext(ctx, checkQuery, user.Email).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Now insert
	query := `
		INSERT INTO users (email, password)
		VALUES ($1, $2)
		RETURNING id, email, password, created_at, updated_at
	`

	usr := User{}
	err = r.db.QueryRowContext(ctx, query, user.Email, user.Password).
		Scan(&usr.ID, &usr.Email, &usr.Password, &usr.CreatedAt, &usr.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &usr, nil
}

func (r *repository) GetUserWithProfile(ctx context.Context, userID int) (*User, *CompleteProfileRequest, error) {
	query := `
		SELECT 
			u.id, u.email, u.password, u.created_at, u.updated_at, u.is_profile_completed, u.is_subscribed,
			p.verse_pace, p.bible_translation, p.enable_notification,
			p.is_email_notification, p.is_web_notification, p.selected_time, p.username
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
		WHERE u.id = $1
	`

	var (
		user    User
		profile CompleteProfileRequest
	)

	// Handle nullable fields from the profile table
	var (
		versePace           sql.NullString
		bibleTranslation    sql.NullString
		enableNotification  sql.NullBool
		isEmailNotification sql.NullBool
		isWebNotification   sql.NullBool
		selectedTime        sql.NullTime
		userName            sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsProfileCompleted,
		&user.IsSubscribed,
		&versePace,
		&bibleTranslation,
		&enableNotification,
		&isEmailNotification,
		&isWebNotification,
		&selectedTime,
		&userName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, fmt.Errorf("user not found")
		}
		return nil, nil, fmt.Errorf("failed to fetch user with profile: %w", err)
	}

	// Map nullable fields only if valid
	if versePace.Valid {
		profile.VersePace = versePace.String
	}
	if bibleTranslation.Valid {
		profile.BibleTranslation = bibleTranslation.String
	}
	if enableNotification.Valid {
		profile.EnableNotification = enableNotification.Bool
	}
	if isEmailNotification.Valid {
		profile.IsEmailNotification = isEmailNotification.Bool
	}
	if isWebNotification.Valid {
		profile.IsWebNotification = isWebNotification.Bool
	}
	if selectedTime.Valid {
		profile.SelectedTime = selectedTime.Time
	}
	if userName.Valid {
		profile.UserName = userName.String
	}

	return &user, &profile, nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	user := User{}
	query := `SELECT id, email, password, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *repository) UpdateUserProfile(ctx context.Context, userID int, req CompleteProfileRequest) error {
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	err := r.db.QueryRowContext(ctx, checkQuery, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("user with id %d does not exist", userID)
	}

	query := `
		INSERT INTO user_profiles (
			user_id, verse_pace, bible_translation,
			enable_notification, is_email_notification,
			is_web_notification, selected_time, username
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id)
		DO UPDATE SET
			verse_pace = EXCLUDED.verse_pace,
			bible_translation = EXCLUDED.bible_translation,
			enable_notification = EXCLUDED.enable_notification,
			is_email_notification = EXCLUDED.is_email_notification,
			is_web_notification = EXCLUDED.is_web_notification,
			selected_time = EXCLUDED.selected_time,
			updated_at = NOW(),
			username = EXCLUDED.username
	`

	_, err = r.db.ExecContext(ctx, query,
		userID,
		req.VersePace,
		req.BibleTranslation,
		req.EnableNotification,
		req.IsEmailNotification,
		req.IsWebNotification,
		req.SelectedTime,
		req.UserName,
	)
	return err
}

func (r *repository) MarkProfileCompleted(ctx context.Context, userID int) error {
	query := `
		UPDATE users
		SET is_profile_completed = TRUE, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *repository) UpdateUserInspirations(ctx context.Context, userID int, inspirations []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// First, clear existing inspirations
	_, err = tx.ExecContext(ctx, `DELETE FROM user_inspirations WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	// Insert new inspirations
	query := `INSERT INTO user_inspirations (user_id, inspiration) VALUES ($1, $2)`
	for _, inspiration := range inspirations {
		_, err = tx.ExecContext(ctx, query, userID, inspiration)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *repository) GetAllUsersWithVersePace(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			u.id, 
			u.email, 
			COALESCE(p.username, '') AS username, 
			COALESCE(p.verse_pace, '') AS verse_pace, 
			u.last_verse_sent_at,
			u.is_subscribed
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Email, &u.UserName, &u.VersePace, &u.LastVerseSentAt, &u.IsSubscribed)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
		log.Printf("Row data: email=%s pace=%s lastSent=%v", u.Email, u.VersePace, u.LastVerseSentAt)

	}

	return users, nil
}

func (r *repository) UpdateLastVerseSentAt(ctx context.Context, userID int, t time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET last_verse_sent_at = $1
		WHERE id = $2
	`, t.UTC(), userID)
	return err
}

func (r *repository) UnsubscribeUser(ctx context.Context, userID int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET is_subscribed = NOT is_subscribed
		WHERE id = $1
	`, userID)
	return err
}
