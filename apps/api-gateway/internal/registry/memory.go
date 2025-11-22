package registry

import (
	"context"
)

// MemoryRepository is a non-persistent in-memory repo for dev
type MemoryRepository struct {
	items map[string]*Service
}

func NewMemoryRepository() *MemoryRepository { return &MemoryRepository{items: map[string]*Service{}} }

func (m *MemoryRepository) Init() error { return nil }
func (m *MemoryRepository) LoadEnabled(ctx context.Context) ([]*Service, error) {
	var list []*Service
	for _, s := range m.items {
		if s.Enabled {
			list = append(list, s)
		}
	}
	return list, nil
}
func (m *MemoryRepository) List(ctx context.Context) ([]*Service, error) {
	var list []*Service
	for _, s := range m.items {
		list = append(list, s)
	}
	return list, nil
}
func (m *MemoryRepository) Get(ctx context.Context, id string) (*Service, error) {
	if s, ok := m.items[id]; ok {
		return s, nil
	}
	return nil, sqlErrNotFound
}
func (m *MemoryRepository) Create(ctx context.Context, s *Service) error {
	m.items[s.ID] = s
	return nil
}
func (m *MemoryRepository) Update(ctx context.Context, s *Service) error {
	m.items[s.ID] = s
	return nil
}
func (m *MemoryRepository) Delete(ctx context.Context, id string) error {
	delete(m.items, id)
	return nil
}

var sqlErrNotFound = &notFound{}

type notFound struct{}

func (n *notFound) Error() string { return "not found" }
