package memoryverse

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/taiwoajasa245/memory-verse-api/pkg/config"
)

// StartScheduler runs the verse delivery job on a schedule.
// - In dev: runs every 1 minute.
// - In prod: runs every 24 hours (daily check for users).
func (s *MemoryVerseService) StartScheduler(ctx context.Context) {
	tickerDuration := time.Hour // default for testing (local/dev)

	log.Println("Current time:", time.Now())

	appEnv := config.GetAppEnv()
	if appEnv == "production" {
		tickerDuration = 24 * time.Hour // daily check in prod
	}

	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	log.Printf("MemoryVerse Scheduler started (%s interval)\n", tickerDuration)

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopped gracefully")
			return
		case <-ticker.C:
			s.runVerseDistribution(ctx)
		}
	}
}

// runVerseDistribution checks each user's verse pace and last sent date.
func (s *MemoryVerseService) runVerseDistribution(ctx context.Context) {

	// if err := s.repo.GenerateDailyVerse(ctx); err != nil {
	// 	log.Printf("Failed to generate daily verse: %v", err)
	// 	return
	// } else {
	// 	log.Println("Daily verse generated successfully")
	// }

	users, err := s.authRepo.GetAllUsersWithVersePace(ctx)
	if err != nil {
		log.Printf("Failed to fetch users for verse distribution: %v", err)
		return
	}

	log.Printf("Running verse distribution check for %d users\n", len(users))

	for _, user := range users {

		if !user.IsSubscribed {
			log.Printf("Skipping user %s (unsubscribed)", user.Email)
			continue
		}
		log.Printf("user versePace is: %s", user.VersePace)

		// Determine next send time based on pace
		var sendInterval time.Duration
		switch user.VersePace {
		case "weekly":
			sendInterval = 7 * 24 * time.Hour
		default:
			// default to daily
			sendInterval = 5 * time.Second
		}

		if user.LastVerseSentAt == nil || time.Since(user.LastVerseSentAt.UTC()) >= sendInterval {
			go func(uID int) {
				_, verse, _, _, err := s.GetUserDashboard(ctx, uID)
				if err != nil {
					log.Printf("Skipping user %d: %v", uID, err)
					return
				}

				data := map[string]interface{}{
					"UserName":       user.UserName,
					"Verse":          verse.Verse,
					"Reference":      verse.Reference,
					"Pace":           user.VersePace,
					"DashboardURL":   "https://memoryverse.app/dashboard",
					"UnsubscribeURL": "https://memoryverse.app/unsubscribe",
				}

				subject := fmt.Sprintf("Your %s Memoryverse is", user.VersePace)

				if err := s.mail.SendHTML(user.Email, subject, "verse.html", data); err != nil {
					log.Printf("Failed to send verse to %s: %v", user.Email, err)
					return
				}

				// Update last sent timestamp
				if err := s.authRepo.UpdateLastVerseSentAt(ctx, uID, time.Now()); err != nil {
					log.Printf("Could not update last sent date for %d: %v", uID, err)
				}

				log.Printf("Verse sent to %s (%s)", user.Email, verse.Reference)
			}(user.ID)
		}
	}
}
