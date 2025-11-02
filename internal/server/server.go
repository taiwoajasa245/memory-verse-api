package server

import (
	"context"
	"fmt"
	"log"

	// "log"
	"net/http"

	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/taiwoajasa245/memory-verse-api/internal/auth"
	"github.com/taiwoajasa245/memory-verse-api/internal/database"
	"github.com/taiwoajasa245/memory-verse-api/internal/mail"
	memoryverse "github.com/taiwoajasa245/memory-verse-api/internal/memory_verse"
	"github.com/taiwoajasa245/memory-verse-api/pkg/config"
)

type Server struct {
	port      string
	db        database.Service
	handler   http.Handler
	cfg       *config.Config
	mail      *mail.Mailer
	mvService memoryverse.MemoryVerseService
	cancel    context.CancelFunc
}

// NewServer constructs your app server with all dependencies injected.
func NewServer(db database.Service, cfg *config.Config) *Server {
	stats := db.Health()
	mail := mail.NewMail(
		cfg.SmtpFrom,
		"Memory Verse",
		cfg.SmtpPassword,
		cfg.SmtpHost,
		cfg.SmtpPort,
	)

	fmt.Println("Database Health:", stats)

	if stats["status"] != "up" {
		log.Fatal("Database connection failed")
		return &Server{}
	} else {
		log.Println("Database connection successful")
	}

	authRepo := auth.NewRepository(db)
	memoryVerseRepo := memoryverse.NewMemoryVerseRepo(db)
	mvService := memoryverse.NewMemoryVerseService(memoryVerseRepo, authRepo, mail)

	s := &Server{
		port:      cfg.Port,
		db:        db,
		cfg:       cfg,
		mail:      mail,
		mvService: mvService,
	}

	s.handler = s.RegisterRoutes()
	return s
}

// HTTPServer returns the actual *http.Server instance
func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%s", s.port),
		Handler:      s.handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// StartBackgroundJobs runs scheduled jobs
func (s *Server) StartBackgroundJobs() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// Start Memory Verse scheduler in background
	go s.mvService.StartScheduler(ctx)
	log.Println("MemoryVerse scheduler started")
}

func (s *Server) StopBackgroundJobs() {
	if s.cancel != nil {
		s.cancel()
		log.Println("Background jobs stopped gracefully")
	}
}
