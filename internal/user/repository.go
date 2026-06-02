package user

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository owns all database access for the user domain. It returns typed
// values and never touches HTTP concerns.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Profile loads the four mutable profile columns for a user. The id in the
// returned Profile is the caller-supplied uid (the row's primary key), so the
// response key is always populated even though the columns are nullable.
func (r *Repository) Profile(ctx context.Context, uid string) (*Profile, error) {
	p := Profile{ID: uid}
	err := r.pool.QueryRow(ctx,
		`SELECT name, email, phone, role FROM users WHERE id = $1`, uid,
	).Scan(&p.Name, &p.Email, &p.Phone, &p.Role)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) ListSaved(ctx context.Context, uid string) ([]SavedDestination, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id::text, d.slug, d.name, d.district, d.altitude_m, d.rating, s.saved_at
		FROM saved_destinations s JOIN destinations d ON d.id = s.destination_id
		WHERE s.user_id = $1 ORDER BY s.saved_at DESC
	`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SavedDestination{}
	for rows.Next() {
		var s SavedDestination
		if err := rows.Scan(&s.ID, &s.Slug, &s.Name, &s.District, &s.AltitudeM, &s.Rating, &s.SavedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Save(ctx context.Context, uid, destinationID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO saved_destinations (user_id, destination_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`, uid, destinationID)
	return err
}

func (r *Repository) Unsave(ctx context.Context, uid, destinationID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM saved_destinations WHERE user_id = $1 AND destination_id = $2`, uid, destinationID)
	return err
}

func (r *Repository) ListItineraries(ctx context.Context, uid string) ([]Itinerary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, title, duration, start_date, is_public, share_token, created_at
		FROM itineraries WHERE user_id = $1 ORDER BY created_at DESC
	`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Itinerary{}
	for rows.Next() {
		var it Itinerary
		if err := rows.Scan(&it.ID, &it.Title, &it.Duration, &it.StartDate, &it.IsPublic, &it.ShareToken, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
