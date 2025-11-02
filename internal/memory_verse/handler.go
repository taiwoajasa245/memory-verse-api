package memoryverse

import (
	"encoding/json"
	"net/http"

	"github.com/taiwoajasa245/memory-verse-api/internal/auth"
	"github.com/taiwoajasa245/memory-verse-api/pkg/response"
)

type MemoryVerseHandler struct {
	service MemoryVerseService
}

func NewMemoryVerseHandler(service MemoryVerseService) MemoryVerseHandler {
	return MemoryVerseHandler{service: service}
}

func (h *MemoryVerseHandler) GetDashboardVerseHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not logged in")
		return
	}

	user, verse, notes, histories, err := h.service.GetUserDashboard(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get memory verse", err.Error())
		return
	}

	if notes == nil {
		notes = []UserNotes{}
	}
	if histories == nil {
		histories = []VerseHistory{}
	}

	response.Success(w, map[string]interface{}{
		"user":          user,
		"verse":         verse,
		"notes":         notes,
		"verse_history": histories,
	}, "successfully")
}

func (h *MemoryVerseHandler) UnsubscribeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not logged in")
		return
	}

	err := h.service.ToggleSubscribeUserService(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to unsubscribe", err.Error())
		return
	}

	response.Success(w, "Ok", "successfully")
}

func (h *MemoryVerseHandler) ToggleFavouriteVerseHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not logged in")
		return
	}

	var req AddToFavouriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", err.Error())
		return
	}

	if req.VerseID == 0 {
		response.Error(w, http.StatusBadRequest, "Missing required fields", map[string]string{
			"verse_id": "verse_id is required",
		})
		return
	}

	verseId := AddToFavouriteRequest{
		VerseID: req.VerseID,
	}

	ok, err := h.service.ToggleFavouriteVerseService(r.Context(), userID, verseId.VerseID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to save favourite", err.Error())
		return
	}

	response.Success(w, map[string]bool{
		"is_saved": ok,
	}, "successfully")
}

func (h *MemoryVerseHandler) GetUserFavouriteVersesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not logged in")
		return
	}

	favourites, err := h.service.GetUserFavouriteVersesService(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get user favourite verses", err.Error())
		return
	}

	if favourites == nil {
		favourites = []FavouriteVerse{}
	}

	response.Success(w, favourites, "successfully")
}
