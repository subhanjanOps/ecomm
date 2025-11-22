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
