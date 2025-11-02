package memoryverse

import "time"

type Verse struct {
	ID          int       `json:"id"`
	Reference   string    `json:"reference"`
	Verse       string    `json:"verse"`
	Translation string    `json:"translation"`
	CreatedAt   time.Time `json:"created_at"`
	IsFavourite bool      `json:"is_favourite"`
}

type VerseHistory struct {
	UserID      int       `json:"user_id,omitempty"`
	VerseID     int       `json:"verse_id"`
	DeliveredAt time.Time `json:"delivered_at"`
	Verse       Verse     `json:"verse"`
}

type UserNotes struct {
	ID             int       `json:"id"`
	VerseReference string    `json:"verse_reference"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type FavouriteVerse struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	VerseID   int       `json:"verse_id"`
	Verse     Verse     `json:"verse"`
	CreatedAt time.Time `json:"created_at"`
}

type AddToFavouriteRequest struct {
	VerseID int `json:"verse_id"`
}
