package memoryverse

import (
	"context"
	"database/sql"
	"errors"

	"github.com/taiwoajasa245/memory-verse-api/internal/database"
)

var (
	ErrNotFound       = errors.New("record not found")
	ErrAlreadyExists  = errors.New("record already exists")
	ErrInternalServer = errors.New("internal server error")
)

type MemoryVerseRepo interface {
	GetRandomVerse(ctx context.Context, userID int, translation string) (*Verse, error)
	GetLastDeliveredVerse(ctx context.Context, userID int) (*VerseHistory, error)
	SaveDeliveredVerse(ctx context.Context, userID, verseID int) error
	SaveUserNote(ctx context.Context, userID int, verseRef, content string) error
	GetUserNotes(ctx context.Context, userID int) ([]UserNotes, error)
	GetAllUserVerseHistory(ctx context.Context, userID int) ([]VerseHistory, error)
	ToggleFavouriteVerse(ctx context.Context, userID, verseID int) (bool, error)
	GetUserFavouriteVerses(ctx context.Context, userID int) ([]FavouriteVerse, error)
	IsVerseFavourited(ctx context.Context, userID, verseID int) (bool, error)
}

type repository struct {
	db *sql.DB
}

func NewMemoryVerseRepo(dbService database.Service) MemoryVerseRepo {
	return &repository{db: dbService.DB()}
}

func (r *repository) GetRandomVerse(ctx context.Context, userID int, translation string) (*Verse, error) {
	query := `
		SELECT 
			mv.id, mv.reference, mv.verse, mv.translation, mv.created_at,
			EXISTS (
				SELECT 1 FROM favourite_verses fv 
				WHERE fv.user_id = $1 AND fv.verse_id = mv.id
			) AS is_favourite
		FROM memory_verses mv
		WHERE mv.translation = $2
		ORDER BY RANDOM()
		LIMIT 1
	`

	var v Verse
	err := r.db.QueryRowContext(ctx, query, userID, translation).Scan(
		&v.ID,
		&v.Reference,
		&v.Verse,
		&v.Translation,
		&v.CreatedAt,
		&v.IsFavourite,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, ErrInternalServer
	}
	return &v, nil
}

func (r *repository) GetLastDeliveredVerse(ctx context.Context, userID int) (*VerseHistory, error) {
	query := `
		SELECT uh.user_id, uh.verse_id, uh.delivered_at,
		       mv.id, mv.reference, mv.verse, mv.translation, mv.created_at
		FROM user_verse_history uh
		JOIN memory_verses mv ON mv.id = uh.verse_id
		WHERE uh.user_id = $1
		ORDER BY uh.delivered_at DESC
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, userID)

	h := VerseHistory{}
	err := row.Scan(
		&h.UserID,
		&h.VerseID,
		&h.DeliveredAt,
		&h.Verse.ID,
		&h.Verse.Reference,
		&h.Verse.Verse,
		&h.Verse.Translation,
		&h.Verse.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, ErrInternalServer
	}
	return &h, nil
}

func (r *repository) SaveDeliveredVerse(ctx context.Context, userID, verseID int) error {
	query := `
		INSERT INTO user_verse_history (user_id, verse_id)
		VALUES ($1, $2)
	`
	_, err := r.db.ExecContext(ctx, query, userID, verseID)
	if err != nil {
		return ErrInternalServer
	}
	return nil
}

func (r *repository) SaveUserNote(ctx context.Context, userID int, verseRef, content string) error {
	query := `
		INSERT INTO user_notes (user_id, verse_reference, content)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, query, userID, verseRef, content)
	if err != nil {
		return ErrInternalServer
	}
	return nil
}

func (r *repository) GetUserNotes(ctx context.Context, userID int) ([]UserNotes, error) {
	query := `
		SELECT id, verse_reference, content, created_at, updated_at
		FROM user_notes
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []UserNotes
	for rows.Next() {
		var note UserNotes
		if err := rows.Scan(&note.ID, &note.VerseReference, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, nil
}

func (r *repository) GetAllUserVerseHistory(ctx context.Context, userID int) ([]VerseHistory, error) {
	query := `
		SELECT uh.verse_id, uh.delivered_at,
		       mv.id, mv.reference, mv.verse, mv.translation, mv.created_at
		FROM user_verse_history uh
		JOIN memory_verses mv ON mv.id = uh.verse_id
		WHERE uh.user_id = $1
		ORDER BY uh.delivered_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, ErrInternalServer
	}
	defer rows.Close()

	var histories []VerseHistory
	for rows.Next() {
		var h VerseHistory
		if err := rows.Scan(
			// &h.UserID,
			&h.VerseID,
			&h.DeliveredAt,
			&h.Verse.ID,
			&h.Verse.Reference,
			&h.Verse.Verse,
			&h.Verse.Translation,
			&h.Verse.CreatedAt,
		); err != nil {
			return nil, ErrInternalServer
		}
		histories = append(histories, h)
	}

	if err = rows.Err(); err != nil {
		return nil, ErrInternalServer
	}

	return histories, nil
}

func (r *repository) ToggleFavouriteVerse(ctx context.Context, userID, verseID int) (bool, error) {
	queryCheck := `
		SELECT EXISTS (
			SELECT 1 FROM favourite_verses WHERE user_id = $1 AND verse_id = $2
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, queryCheck, userID, verseID).Scan(&exists)
	if err != nil {
		return false, ErrNotFound
	}

	if exists {

		_, err = r.db.ExecContext(ctx, `
			DELETE FROM favourite_verses WHERE user_id = $1 AND verse_id = $2
		`, userID, verseID)
		if err != nil {
			return false, ErrInternalServer
		}
		return false, nil
	}

	// Otherwise, add it
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO favourite_verses (user_id, verse_id)
		VALUES ($1, $2)
	`, userID, verseID)
	if err != nil {
		return false, ErrInternalServer
	}

	return true, nil // now favourited
}

func (r *repository) GetUserFavouriteVerses(ctx context.Context, userID int) ([]FavouriteVerse, error) {
	query := `
		SELECT fv.id, fv.user_id, fv.verse_id, fv.created_at,
		       mv.id, mv.reference, mv.verse, mv.translation, mv.created_at
		FROM favourite_verses fv
		JOIN memory_verses mv ON mv.id = fv.verse_id
		WHERE fv.user_id = $1
		ORDER BY fv.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favourites []FavouriteVerse
	for rows.Next() {
		var fav FavouriteVerse
		err := rows.Scan(
			&fav.ID, &fav.UserID, &fav.VerseID, &fav.CreatedAt,
			&fav.Verse.ID, &fav.Verse.Reference, &fav.Verse.Verse,
			&fav.Verse.Translation, &fav.Verse.CreatedAt,
		)
		if err != nil {
			return nil, ErrInternalServer
		}
		favourites = append(favourites, fav)
	}

	return favourites, nil
}

func (r *repository) IsVerseFavourited(ctx context.Context, userID, verseID int) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM favourite_verses WHERE user_id = $1 AND verse_id = $2
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID, verseID).Scan(&exists)
	if err != nil {
		return false, ErrInternalServer
	}
	return exists, err
}
