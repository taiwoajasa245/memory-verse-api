package memoryverse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/taiwoajasa245/memory-verse-api/internal/auth"
	"github.com/taiwoajasa245/memory-verse-api/internal/mail"
)

type MemoryVerseService struct {
	repo     MemoryVerseRepo
	authRepo auth.Repository
	mail     *mail.Mailer
}

func NewMemoryVerseService(repo MemoryVerseRepo, authRepo auth.Repository, mail *mail.Mailer) MemoryVerseService {
	return MemoryVerseService{
		repo:     repo,
		authRepo: authRepo,
		mail:     mail,
	}
}

func (s *MemoryVerseService) SendDailyVerses(ctx context.Context) (*Verse, error) {
	verse, err := s.repo.GetDailyVerse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily verse: %w", err)
	}
	return verse, nil
}

func (s *MemoryVerseService) GetUserDashboard(ctx context.Context, userID int) (*auth.User, *Verse, []UserNotes, []VerseHistory, error) {
	user, profile, err := s.authRepo.GetUserWithProfile(ctx, userID)
	if err != nil {
		log.Printf("error fetching user: %v", err)
		return nil, nil, nil, nil, errors.New("user not found")
	}

	if !user.IsProfileCompleted {
		return nil, nil, nil, nil, errors.New("please complete your profile to receive memory verses")
	}

	pace := strings.ToLower(user.VersePace)
	if pace != "daily" && pace != "weekly" {
		return nil, nil, nil, nil, fmt.Errorf("invalid verse pace: %s", pace)
	}

	lastDelivered, err := s.repo.GetLastDeliveredVerse(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("error fetching last delivered: %v", err)
		// return nil, nil, nil, nil, fmt.Errorf("database error during last delivered fetch: %w", err)
	}

	fmt.Println("last delivered: ", lastDelivered)

	// Determine if a new verse *needs* to be fetched
	now := time.Now()
	shouldSend := false

	switch pace {
	case "daily":
		shouldSend = lastDelivered == nil || now.Sub(lastDelivered.DeliveredAt).Hours() >= 24
	case "weekly":
		shouldSend = lastDelivered == nil || now.Sub(lastDelivered.DeliveredAt).Hours() >= 168
	}

	// NOTE: 'shouldSend' now means "time to get a *NEW* verse from the database".

	notes, err := s.repo.GetUserNotes(ctx, userID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get user notes: %w", err)
	}

	histories, err := s.repo.GetAllUserVerseHistory(ctx, userID)
	if err != nil {
		log.Printf("failed to get user verse history: %v", err)
		return nil, nil, nil, nil, fmt.Errorf("failed to get user verse history: %w", err)
	}

	// If shouldSend, fetch a new verse and save it
	if shouldSend {
		verse, err := s.repo.GetRandomVerse(ctx, userID, profile.BibleTranslation)
		if err != nil {
			log.Printf("error fetching random verse: %v", err)

			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil, nil, nil, fmt.Errorf("no verses found in database for translation %s", profile.BibleTranslation)
			}

			return nil, nil, nil, nil, err
		}

		_ = s.repo.SaveDeliveredVerse(ctx, userID, verse.ID)
		return user, verse, notes, histories, nil
	} else {
		// It is NOT time to send a new verse, so we return the *last* one we found.
		if lastDelivered != nil {
			return user, &lastDelivered.Verse, notes, histories, nil
		}

		// This case should ideally not be hit if initial user setup ensures a first verse is sent immediately,
		// but it's the final fallback.
		return user, nil, notes, histories, fmt.Errorf("internal logic error: should have either sent new verse or returned existing one")
	}
}

func (s *MemoryVerseService) ToggleSubscribeUserService(ctx context.Context, userID int) error {
	return s.authRepo.UnsubscribeUser(ctx, userID)
}

func (s *MemoryVerseService) ToggleFavouriteVerseService(ctx context.Context, userID int, verseID int) (bool, error) {

	isFav, err := s.repo.ToggleFavouriteVerse(ctx, userID, verseID)
	if err != nil {
		log.Println("Error toggling favourite:", err)
		return false, err
	}

	return isFav, nil
}

func (s *MemoryVerseService) GetUserFavouriteVersesService(ctx context.Context, userID int) ([]FavouriteVerse, error) {
	favourites, err := s.repo.GetUserFavouriteVerses(ctx, userID)
	if err != nil {
		log.Println("Error fetching user favourites:", err)
		return nil, err
	}

	return favourites, nil
}

func (s *MemoryVerseService) SaveUserNote(ctx context.Context, userId int, content, verse_ref string) error {

	if err := s.repo.SaveUserNote(ctx, userId, verse_ref, content); err != nil {
		log.Println("Error saving user notes:", err)
		return err
	}

	return nil
}
