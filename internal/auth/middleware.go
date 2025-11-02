package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/taiwoajasa245/memory-verse-api/pkg/response"
	"github.com/taiwoajasa245/memory-verse-api/pkg/util"
)

type contextKey string

const (
	userContextKey   contextKey = "user"
	userIDContextKey contextKey = "user_id"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			response.Error(w, http.StatusUnauthorized, "Missing Authorization header", "user not logged in")
			return
		}

		// Must start with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			// http.Error(w, "Invalid token format", http.StatusUnauthorized)
			response.Error(w, http.StatusUnauthorized, "Invalid token format", "")

			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := util.ValidateJWT(tokenStr)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, claims)
		ctx = context.WithValue(ctx, userIDContextKey, claims.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func GetUserFromContext(r *http.Request) (*util.Claims, bool) {
	claims, ok := r.Context().Value(userContextKey).(*util.Claims)
	return claims, ok
}

func GetUserIDFromContext(r *http.Request) (int, bool) {
	id, ok := r.Context().Value(userIDContextKey).(int)
	return id, ok
}
