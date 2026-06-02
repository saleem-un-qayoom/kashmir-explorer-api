package destination

import "context"

// Service holds the destination domain's business logic. Most methods are thin
// pass-throughs today; the seam lets validation/orchestration grow without
// touching the HTTP or data layers.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ─── Public ─────────────────────────────────────────────────────

func (s *Service) List(ctx context.Context, region, category string, limit, offset int) ([]Destination, error) {
	return s.repo.List(ctx, region, category, limit, offset)
}

func (s *Service) Featured(ctx context.Context) ([]FeaturedDestination, error) {
	return s.repo.Featured(ctx)
}

func (s *Service) Trending(ctx context.Context) ([]TrendingDestination, error) {
	return s.repo.Trending(ctx)
}

func (s *Service) Nearby(ctx context.Context, lng, lat, radius float64, limit int) ([]NearbyDestination, error) {
	return s.repo.Nearby(ctx, lng, lat, radius, limit)
}

func (s *Service) Bbox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]MapPin, error) {
	return s.repo.Bbox(ctx, minLng, minLat, maxLng, maxLat)
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*Destination, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	return s.repo.Categories(ctx)
}

func (s *Service) Regions(ctx context.Context) ([]Region, error) {
	return s.repo.Regions(ctx)
}

// ─── Admin: destinations ────────────────────────────────────────

func (s *Service) AdminList(ctx context.Context, status string) ([]AdminDest, error) {
	return s.repo.AdminList(ctx, status)
}

func (s *Service) AdminGet(ctx context.Context, id string) (*AdminDest, error) {
	return s.repo.AdminGet(ctx, id)
}

func (s *Service) AdminCreate(ctx context.Context, in AdminDestInput) (string, error) {
	return s.repo.AdminCreate(ctx, in)
}

func (s *Service) AdminUpdate(ctx context.Context, id string, in AdminDestInput) error {
	return s.repo.AdminUpdate(ctx, id, in)
}

func (s *Service) AdminSoftDelete(ctx context.Context, id string) error {
	return s.repo.AdminSoftDelete(ctx, id)
}

func (s *Service) AdminRestore(ctx context.Context, id string) error {
	return s.repo.AdminRestore(ctx, id)
}

func (s *Service) AdminDeletePermanent(ctx context.Context, id string) error {
	return s.repo.AdminDeletePermanent(ctx, id)
}

// ─── Admin: categories ──────────────────────────────────────────

func (s *Service) CategoryGet(ctx context.Context, id string) (*Category, error) {
	return s.repo.CategoryGet(ctx, id)
}

func (s *Service) CategoryCreate(ctx context.Context, in CategoryInput) (string, error) {
	return s.repo.CategoryCreate(ctx, in)
}

func (s *Service) CategoryUpdate(ctx context.Context, id string, in CategoryInput) error {
	return s.repo.CategoryUpdate(ctx, id, in)
}

func (s *Service) CategoryDelete(ctx context.Context, id string) error {
	return s.repo.CategoryDelete(ctx, id)
}

// ─── Admin: regions ─────────────────────────────────────────────

func (s *Service) RegionGet(ctx context.Context, id string) (*Region, error) {
	return s.repo.RegionGet(ctx, id)
}

func (s *Service) RegionCreate(ctx context.Context, in RegionInput) (string, error) {
	return s.repo.RegionCreate(ctx, in)
}

func (s *Service) RegionUpdate(ctx context.Context, id string, in RegionInput) error {
	return s.repo.RegionUpdate(ctx, id, in)
}

func (s *Service) RegionDelete(ctx context.Context, id string) error {
	return s.repo.RegionDelete(ctx, id)
}
