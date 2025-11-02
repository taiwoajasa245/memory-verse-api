package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/taiwoajasa245/memory-verse-api/internal/auth"
	memoryverse "github.com/taiwoajasa245/memory-verse-api/internal/memory_verse"
	"github.com/taiwoajasa245/memory-verse-api/pkg/response"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Get home route
	r.Get("/", s.ServerIsWorking)
	r.Get("/memory-verse-api/v1", s.ServerIsWorking)

	r.Route("/memory-verse-api/v1", func(r chi.Router) {
		s.loadAuthRoutes(r)
		s.loadVerseRoutes(r)
	})

	return r
}

func (s *Server) ServerIsWorking(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Welcome to Memory verse api"

	response.Success(w, resp, "Success")
}

func (s *Server) loadAuthRoutes(router chi.Router) {

	authRepo := auth.NewRepository(s.db)
	authServie := auth.NewAuthService(authRepo, s.mail)
	authHandler := auth.NewHandler(authServie)

	router.Post("/auth/login", authHandler.LoginHandler)
	router.Post("/auth/register-with-email", authHandler.RegisterHandler)

	router.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Post("/auth/complete-profile", authHandler.CompleteProfileHandler)
	})

}

func (s *Server) loadVerseRoutes(router chi.Router) {
	authRepo := auth.NewRepository(s.db)
	memoryVerseRepo := memoryverse.NewMemoryVerseRepo(s.db)
	memeoryVerseService := memoryverse.NewMemoryVerseService(memoryVerseRepo, authRepo, s.mail)
	memeoryVerseHandler := memoryverse.NewMemoryVerseHandler(memeoryVerseService)

	router.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Get("/dashboard", memeoryVerseHandler.GetDashboardVerseHandler)
		r.Get("/unsubscribe", memeoryVerseHandler.UnsubscribeHandler)
		r.Get("/get-favourite-verses", memeoryVerseHandler.GetUserFavouriteVersesHandler)
		r.Patch("/toggle-favourite-verse", memeoryVerseHandler.ToggleFavouriteVerseHandler)
	})

}
