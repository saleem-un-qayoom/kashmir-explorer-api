// Package user — current-user profile, saved destinations, and itineraries.
//
// Layering: routes.go → handler.go → service.go → repository.go, with the
// transport-agnostic data shapes defined here in dto.go. JSON tags mirror the
// keys the pre-refactor handlers emitted, byte-for-byte.
package user

import "time"

// Profile is the response for GET /v1/me. Fields are pointers because the
// underlying columns are nullable; the keys are always present (no omitempty)
// to match the previous map[string]any payload exactly.
type Profile struct {
	ID    string  `json:"id"`
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
	Role  *string `json:"role"`
}

// SavedDestination is one row of GET /v1/saved.
type SavedDestination struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	District  *string   `json:"district"`
	AltitudeM *int      `json:"altitude_m"`
	Rating    float64   `json:"rating"`
	SavedAt   time.Time `json:"saved_at"`
}

// Itinerary is one row of GET /v1/itineraries.
type Itinerary struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Duration   *int       `json:"duration"`
	StartDate  *time.Time `json:"start_date"`
	IsPublic   bool       `json:"is_public"`
	ShareToken *string    `json:"share_token"`
	CreatedAt  time.Time  `json:"created_at"`
}
