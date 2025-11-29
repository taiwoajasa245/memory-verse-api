package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/taiwoajasa245/memory-verse-api/docs"

	"github.com/taiwoajasa245/memory-verse-api/internal/auth"
	memoryverse "github.com/taiwoajasa245/memory-verse-api/internal/memory_verse"
	"github.com/taiwoajasa245/memory-verse-api/pkg/response"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// r.Use(middleware.RedirectSlashes)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Get home route
	r.Get("/", s.ServerIsWorking)

	// Redirect root to Swagger
	r.Get("/memory-verse-api/v1/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})

	// Serve swagger files from swaggo/files
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	r.Route("/memory-verse-api/v1", func(r chi.Router) {
		s.loadAuthRoutes(r)
		s.loadVerseRoutes(r)
	})
	r.Get("/memory-verse-api/v1", s.ServerIsWorking)

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
	router.Post("/auth/forget-password", authHandler.ForgetPasswordHandler)
	router.Post("/auth/reset-password", authHandler.ResetPasswordHandler)

	router.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Get("/auth/me", authHandler.GetUserDetailsHandler)
		r.Post("/auth/complete-profile", authHandler.CompleteProfileHandler)
		r.Get("/auth/verify-token", authHandler.VerifyTokenHandler)
		r.Patch("/auth/update-profile", authHandler.UpdateUserProfileHandler)
	})

}

func (s *Server) loadVerseRoutes(router chi.Router) {
	authRepo := auth.NewRepository(s.db)
	memoryVerseRepo := memoryverse.NewMemoryVerseRepo(s.db)
	memeoryVerseService := memoryverse.NewMemoryVerseService(memoryVerseRepo, authRepo, s.mail)
	memeoryVerseHandler := memoryverse.NewMemoryVerseHandler(memeoryVerseService)

	router.Group(
		func(r chi.Router) {
			r.Get("/memoryverse/daily-verse", memeoryVerseHandler.GetDailyVerseHandler)
		},
	)

	router.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Get("/memoryverse/dashboard", memeoryVerseHandler.GetDashboardVerseHandler)
		r.Get("/memoryverse/unsubscribe", memeoryVerseHandler.UnsubscribeHandler)
		r.Get("/memoryverse/get-favourite-verses", memeoryVerseHandler.GetUserFavouriteVersesHandler)
		r.Patch("/memoryverse/toggle-favourite-verse", memeoryVerseHandler.ToggleFavouriteVerseHandler)
		r.Post("/memoryverse/save-note", memeoryVerseHandler.SaveUserNoteHandler)
	})

}
