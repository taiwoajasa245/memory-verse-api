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

// GetDashboardVerseHandler godoc
// @Summary Get user dashboard data
// @Description Retrieve user's memory verse, notes, and history
// @Tags MemoryVerse
// @Produce  json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /memoryverse/dashboard [get]
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

// UnsubscribeHandler godoc
// @Summary Unsubscribe or re-subscribe user
// @Description Toggle user subscription status for verse delivery
// @Tags MemoryVerse
// @Produce  json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /memoryverse/unsubscribe [get]
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

// ToggleFavouriteVerseHandler godoc
// @Summary Add or remove a verse from favourites
// @Description Toggle favourite verse status for logged-in user
// @Tags MemoryVerse
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param   request body AddToFavouriteRequest true "Verse ID to toggle"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /memoryverse/toggle-favourite-verse [patch]
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

// GetUserFavouriteVersesHandler godoc
// @Summary Get user's favourite verses
// @Description Retrieve all verses that the user has marked as favourites
// @Tags MemoryVerse
// @Produce  json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /memoryverse/get-favourite-verses [get]
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


// GetDailyVerseHandler godoc
// @Summary Get Dailly verses
// @Description Retrieve random verse
// @Tags MemoryVerse
// @Produce  json
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /memoryverse/daily-verse [get]
func (h *MemoryVerseHandler) GetDailyVerseHandler(w http.ResponseWriter, r *http.Request) {

	verse, err := h.service.SendDailyVerses(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get daily verse", err.Error())
		return
	}

	if verse == nil {
		verse = &Verse{}
		return
	}

	response.Success(w, verse, "successfully")
}


// SaveUserNoteHandler godoc
// @Summary Get user's Verses Note
// @Description Save user verse note
// @Tags MemoryVerse
// @Produce  json
// @Security BearerAuth
// @Param   request body SaveUserNoteRequest true "Save user note request"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /memoryverse/save-note [post]
func (h *MemoryVerseHandler) SaveUserNoteHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not logged in")
		return
	}

	var req SaveUserNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", err.Error())
		return
	}

	if req.VerseReference == "" || req.Content == "" {
		response.Error(w, http.StatusBadRequest, "Missing required fields", map[string]string{
			"verse_reference": "verse_reference is required",
			"content":         "content is required",
		})
		return
	}

	err := h.service.SaveUserNote(r.Context(), userID, req.Content, req.VerseReference)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to save user note", err.Error())
		return 
	}


	response.Success(w, "Ok", "successfully")
}