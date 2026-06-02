package user

import "context"

// Service holds the user domain's business logic. Today the logic is thin and
// mostly delegates to the repository, but the seam exists so that validation,
// authorization, and orchestration can be added without touching the HTTP or
// data layers.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Profile(ctx context.Context, uid string) (*Profile, error) {
	return s.repo.Profile(ctx, uid)
}

func (s *Service) ListSaved(ctx context.Context, uid string) ([]SavedDestination, error) {
	return s.repo.ListSaved(ctx, uid)
}

func (s *Service) Save(ctx context.Context, uid, destinationID string) error {
	return s.repo.Save(ctx, uid, destinationID)
}

func (s *Service) Unsave(ctx context.Context, uid, destinationID string) error {
	return s.repo.Unsave(ctx, uid, destinationID)
}

func (s *Service) ListItineraries(ctx context.Context, uid string) ([]Itinerary, error) {
	return s.repo.ListItineraries(ctx, uid)
}
