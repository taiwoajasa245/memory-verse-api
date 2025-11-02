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

func (s *MemoryVerseService) GetUserDashboard(ctx context.Context, userID int) (*auth.User, *Verse, []UserNotes, []VerseHistory, error) {
	user, profile, err := s.authRepo.GetUserWithProfile(ctx, userID)
	if err != nil {
		log.Printf("error fetching user: %v", err)
		return nil, nil, nil, nil, errors.New("user not found")
	}

	if !user.IsProfileCompleted {
		return nil, nil, nil, nil, errors.New("please complete your profile to receive memory verses")
	}

	pace := strings.ToLower(profile.VersePace)
	if pace != "daily" && pace != "weekly" {
		return nil, nil, nil, nil, fmt.Errorf("invalid verse pace: %s", pace)
	}

	lastDelivered, err := s.repo.GetLastDeliveredVerse(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("error fetching last delivered: %v", err)
		return nil, nil, nil, nil, err
	}

	fmt.Println("last delivered: ", lastDelivered)

	now := time.Now()
	shouldSend := false

	switch pace {
	case "daily":
		shouldSend = lastDelivered == nil || now.Sub(lastDelivered.DeliveredAt).Hours() <= 24
	case "weekly":
		shouldSend = lastDelivered == nil || now.Sub(lastDelivered.DeliveredAt).Hours() >= 168
	}

	// Always load user notes once
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
			return nil, nil, nil, nil, err
		}

		// record that we sent it
		_ = s.repo.SaveDeliveredVerse(ctx, userID, verse.ID)
		return user, verse, notes, histories, nil
	}

	// otherwise return last one
	if lastDelivered != nil {
		return user, &lastDelivered.Verse, notes, histories, nil
	}

	return user, nil, notes, histories, fmt.Errorf("no verse available")
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
