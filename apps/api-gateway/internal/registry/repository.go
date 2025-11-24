package registry

import (
	"context"
)

// Repository abstracts persistence for services
type Repository interface {
	Init() error
	LoadEnabled(ctx context.Context) ([]*Service, error)
	List(ctx context.Context) ([]*Service, error)
	Get(ctx context.Context, id string) (*Service, error)
	Create(ctx context.Context, s *Service) error
	Update(ctx context.Context, s *Service) error
	Delete(ctx context.Context, id string) error

	// Route mappings for REST -> gRPC transcoding
	ListRoutes(ctx context.Context, serviceID string) ([]*Route, error)
	GetRoute(ctx context.Context, serviceID, routeID string) (*Route, error)
	CreateRoute(ctx context.Context, r *Route) error
	UpdateRoute(ctx context.Context, r *Route) error
	DeleteRoute(ctx context.Context, serviceID, routeID string) error
	FindRoute(ctx context.Context, serviceID, method, path string) (*Route, error)
}

// LoadEnabled loads enabled services into runtime registry
func LoadEnabled(repo Repository, reg *Registry) error {
	list, err := repo.LoadEnabled(context.Background())
	if err != nil {
		return err
	}
	reg.Set(list)
	return nil
}
