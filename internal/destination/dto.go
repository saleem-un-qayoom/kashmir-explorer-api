// Package destination — destinations CRUD + search + nearby + bbox, plus the
// category/region taxonomy.
//
// Layering: routes.go → handler.go → service.go → repository.go, with all
// transport-agnostic data shapes defined here in dto.go. JSON tags mirror the
// keys the pre-refactor handlers emitted; the public list/featured/etc. payloads
// were previously built from map[string]any and are now typed structs with the
// same keys (no omitempty on keys that always serialized before).
package destination

import "encoding/json"

// Destination is the full public detail shape (GET /v1/destinations,
// GET /v1/destinations/{slug}).
type Destination struct {
	ID                   string   `json:"id"`
	Slug                 string   `json:"slug"`
	Name                 string   `json:"name"`
	NameUrdu             *string  `json:"name_urdu,omitempty"`
	NameHindi            *string  `json:"name_hindi,omitempty"`
	District             *string  `json:"district,omitempty"`
	Tagline              *string  `json:"tagline,omitempty"`
	Uniqueness           *string  `json:"uniqueness,omitempty"`
	Lat                  float64  `json:"lat"`
	Lng                  float64  `json:"lng"`
	AltitudeM            *int     `json:"altitude_m,omitempty"`
	BestMonths           []int    `json:"best_months,omitempty"`
	SeasonType           *string  `json:"season_type,omitempty"`
	Rating               float64  `json:"rating"`
	ReviewCount          int      `json:"review_count"`
	DistanceFromSrinagar *int     `json:"distance_from_srinagar_km,omitempty"`
	EntryFeeINR          int      `json:"entry_fee_inr"`
	HasEntryFee          bool     `json:"has_entry_fee"`
	Permits              []string `json:"permits,omitempty"`
	RequiresPermit       bool     `json:"requires_permit"`
	Categories           []string `json:"categories,omitempty"`
	Features             []string `json:"features,omitempty"`
	Description          *string  `json:"description,omitempty"`
	HeroImageURL         *string  `json:"hero_image_url,omitempty"`
}

// FeaturedDestination is one row of GET /v1/destinations/featured.
type FeaturedDestination struct {
	ID         string  `json:"id"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	Tagline    *string `json:"tagline"`
	Uniqueness *string `json:"uniqueness"`
	AltitudeM  *int    `json:"altitude_m"`
	Rating     float64 `json:"rating"`
}

// TrendingDestination is one row of GET /v1/destinations/trending.
type TrendingDestination struct {
	ID                   string  `json:"id"`
	Slug                 string  `json:"slug"`
	Name                 string  `json:"name"`
	Tagline              *string `json:"tagline"`
	Uniqueness           *string `json:"uniqueness"`
	AltitudeM            *int    `json:"altitude_m"`
	Rating               float64 `json:"rating"`
	District             *string `json:"district"`
	DistanceFromSrinagar *int    `json:"distance_from_srinagar_km"`
	HeroImageURL         *string `json:"hero_image_url"`
}

// NearbyDestination is one row of GET /v1/destinations/nearby.
type NearbyDestination struct {
	ID         string  `json:"id"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	District   string  `json:"district"`
	AltitudeM  *int    `json:"altitude_m"`
	Rating     float64 `json:"rating"`
	DistanceKm float64 `json:"distance_km"`
}

// MapPin is one row of GET /v1/destinations/map (bbox viewport pins).
type MapPin struct {
	ID           string   `json:"id"`
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	Lng          float64  `json:"lng"`
	Lat          float64  `json:"lat"`
	Categories   []string `json:"categories"`
	HeroImageURL *string  `json:"hero_image_url"`
}

// Category is the public/admin shape for GET /v1/categories[/{id}].
type Category struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Slug  string  `json:"slug"`
	Icon  *string `json:"icon"`
	Color *string `json:"color"`
}

// Region is the public/admin shape for GET /v1/regions[/{id}].
type Region struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}

// AdminDest is the full admin shape (includes unpublished/deleted rows and the
// extra editorial fields not exposed publicly).
type AdminDest struct {
	ID              string          `json:"id"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	NameUrdu        *string         `json:"name_urdu"`
	NameHindi       *string         `json:"name_hindi"`
	District        *string         `json:"district"`
	RegionSlug      *string         `json:"region_slug"`
	Tagline         *string         `json:"tagline"`
	Uniqueness      *string         `json:"uniqueness"`
	Description     *string         `json:"description"`
	Lat             float64         `json:"lat"`
	Lng             float64         `json:"lng"`
	AltitudeM       *int            `json:"altitude_m"`
	BestMonths      []int           `json:"best_months"`
	SeasonType      *string         `json:"season_type"`
	Rating          float64         `json:"rating"`
	ReviewCount     int             `json:"review_count"`
	DistFromSgr     *int            `json:"distance_from_srinagar_km"`
	EntryFee        int             `json:"entry_fee_inr"`
	HasEntryFee     bool            `json:"has_entry_fee"`
	Permits         []string        `json:"permits"`
	RequiresPermit  bool            `json:"requires_permit"`
	Activities      []string        `json:"activities"`
	NetworkCoverage json.RawMessage `json:"network_coverage"`
	Practical       json.RawMessage `json:"practical"`
	Categories      []string        `json:"categories"`
	IsPublished     bool            `json:"is_published"`
	IsFeatured      bool            `json:"is_featured"`
	IsDeleted       bool            `json:"is_deleted"`
	Features        []string        `json:"features"` // AllTrails-style tags
}

// AdminDestInput is the create/update request body for a destination.
type AdminDestInput struct {
	Name            string          `json:"name"`
	NameUrdu        *string         `json:"name_urdu"`
	NameHindi       *string         `json:"name_hindi"`
	Slug            string          `json:"slug"`
	RegionSlug      string          `json:"region_slug"`
	District        *string         `json:"district"`
	Tagline         *string         `json:"tagline"`
	Uniqueness      *string         `json:"uniqueness"`
	Lat             float64         `json:"lat"`
	Lng             float64         `json:"lng"`
	AltitudeM       *int            `json:"altitude_m"`
	BestMonths      []int           `json:"best_months"`
	SeasonType      *string         `json:"season_type"`
	DistFromSgr     *int            `json:"distance_from_srinagar_km"`
	EntryFee        int             `json:"entry_fee_inr"`
	HasEntryFee     bool            `json:"has_entry_fee"`
	Permits         []string        `json:"permits"`
	RequiresPermit  bool            `json:"requires_permit"`
	Activities      []string        `json:"activities"`
	NetworkCoverage json.RawMessage `json:"network_coverage"`
	Practical       json.RawMessage `json:"practical"`
	Categories      []string        `json:"categories"`
	IsPublished     bool            `json:"is_published"`
	IsFeatured      bool            `json:"is_featured"`
	Description     *string         `json:"description"`
	Features        []string        `json:"features"` // AllTrails-style tags (migration 0010)
}

// CategoryInput is the create/update body for a category.
type CategoryInput struct {
	Name  string  `json:"name"`
	Slug  string  `json:"slug"`
	Icon  *string `json:"icon"`
	Color *string `json:"color"`
}

// RegionInput is the create/update body for a region.
type RegionInput struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}
