package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	"github.com/taiwoajasa245/memory-verse-api/internal/database"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInternalServer     = errors.New("internal server error")
)

// Repository defines the methods the Auth module provides for DB operations.
type Repository interface {
	CreateUser(ctx context.Context, user User) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CompleteUserProfile(ctx context.Context, userID int, req CompleteProfileRequest) error
	MarkProfileCompleted(ctx context.Context, userID int) error
	UpdateUserInspirations(ctx context.Context, userID int, inspirations []string) error
	GetUserWithProfile(ctx context.Context, userID int) (*User, *CompleteProfileRequest, error)
	GetAllUsers(ctx context.Context) ([]User, error)
	GetAllUsersWithVersePace(ctx context.Context) ([]User, error)
	UpdateLastVerseSentAt(ctx context.Context, userID int, t time.Time) error
	UnsubscribeUser(ctx context.Context, userID int) error
	UpdateUserProfile(ctx context.Context, userID int, req UpdateUserProfileRequest) error

	GetUserDetails(ctx context.Context, userId int) (*UserDetails, error)

	SavePasswordReset(ctx context.Context, email, otp string, expiresAt time.Time) error
	GetPasswordReset(ctx context.Context, email string) (string, time.Time, error)
	DeletePasswordReset(ctx context.Context, email string) error
	UpdateUserPassword(ctx context.Context, email, hashed string) error
}

type repository struct {
	db *sql.DB
}

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
		user.VersePace = versePace.String
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
		user.UserName = userName.String
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

func (r *repository) GetUserDetails(ctx context.Context, userId int) (*UserDetails, error) {
	query := `
        SELECT
            u.id,
            u.email,
            u.created_at,
            u.updated_at,
			u.is_profile_completed, 
			u.is_subscribed,
            up.username, -- Maps to UserDetails.UserName
            up.verse_pace,
            up.bible_translation,
            up.enable_notification,
            up.is_email_notification,
            up.is_web_notification,
            up.selected_time,
			ARRAY_REMOVE(ARRAY_AGG(ui.inspiration), NULL) AS inspirations
        FROM
            users u
        LEFT JOIN
            user_profiles up ON u.id = up.user_id
		LEFT JOIN
			user_inspirations ui ON u.id = ui.user_id
        WHERE
            u.id = $1
		GROUP BY
            u.id, up.username, up.verse_pace, up.bible_translation, up.enable_notification, 
            up.is_email_notification, up.is_web_notification, up.selected_time
    `

	details := UserDetails{}

	var inspirationsArray pq.StringArray

	err := r.db.QueryRowContext(ctx, query, userId).Scan(
		&details.ID,
		&details.Email,
		&details.CreatedAt,
		&details.UpdatedAt,
		&details.IsProfileCompleted,
		&details.IsSubscribed,
		&details.UserName,
		&details.VersePace,
		&details.BibleTranslation,
		&details.EnableNotification,
		&details.IsEmailNotification,
		&details.IsWebNotification,
		&details.SelectedTime,
		&inspirationsArray,
	)

	if err != nil {
		// Handle no rows found specifically
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with ID %d not found", userId)
		}
		return nil, fmt.Errorf("failed to retrieve user details: %w", err)
	}

	details.Inspirations = []string(inspirationsArray)

	return &details, nil
}

func (r *repository) UpdateUserProfile(ctx context.Context, userID int, req UpdateUserProfileRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback()

	updateUserQuery := `
		UPDATE users
		SET email = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err = tx.ExecContext(ctx, updateUserQuery, req.Email, userID)
	if err != nil {
		return fmt.Errorf("failed to update user email: %w", err)
	}

	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`

	err = tx.QueryRowContext(ctx, checkQuery, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("user with id %d does not exist", userID)
	}

	upsertProfileQuery := `
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

	_, err = tx.ExecContext(ctx, upsertProfileQuery,
		userID,
		req.VersePace,
		req.BibleTranslation,
		req.EnableNotification,
		req.IsEmailNotification,
		req.IsWebNotification,
		req.SelectedTime,
		req.UserName,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert user profile: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *repository) CompleteUserProfile(ctx context.Context, userID int, req CompleteProfileRequest) error {
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

func (r *repository) SavePasswordReset(ctx context.Context, email, otp string, expiresAt time.Time) error {
	query := `
		INSERT INTO password_resets (email, otp, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (email)
		DO UPDATE SET otp = EXCLUDED.otp, expires_at = EXCLUDED.expires_at
	`

	_, err := r.db.ExecContext(ctx, query, email, otp, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("failed to save password reset: %w", err)
	}
	return nil
}

func (r *repository) GetPasswordReset(ctx context.Context, email string) (string, time.Time, error) {
	var (
		otp       string
		expiresAt time.Time
	)

	query := `SELECT otp, expires_at FROM password_resets WHERE email = $1`

	err := r.db.QueryRowContext(ctx, query, email).Scan(&otp, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Time{}, fmt.Errorf("no password reset record found")
		}
		return "", time.Time{}, fmt.Errorf("failed to fetch password reset: %w", err)
	}

	return otp, expiresAt, nil
}

func (r *repository) DeletePasswordReset(ctx context.Context, email string) error {
	query := `DELETE FROM password_resets WHERE email = $1`
	_, err := r.db.ExecContext(ctx, query, email)
	if err != nil {
		return fmt.Errorf("failed to delete password reset: %w", err)
	}
	return nil
}

func (r *repository) UpdateUserPassword(ctx context.Context, email, hashed string) error {
	query := `
		UPDATE users
		SET password = $1, updated_at = NOW()
		WHERE email = $2
	`

	res, err := r.db.ExecContext(ctx, query, hashed, email)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
