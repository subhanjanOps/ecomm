package registry

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// CachingRepository decorates a Repository with Redis caching for reads.
// Keys:
//   - gateway:services:enabled -> JSON array of services (LoadEnabled)
//   - gateway:services:list    -> JSON array of services (List)
//   - gateway:service:<id>     -> JSON object (Get)
//
// Invalidate on Create/Update/Delete.
type CachingRepository struct {
	inner Repository
	rdb   *redis.Client
	ttl   time.Duration
}

func NewCachingRepository(inner Repository, rdb *redis.Client, ttl time.Duration) *CachingRepository {
	if ttl <= 0 {
		ttl = 15 * time.Second
	}
	return &CachingRepository{inner: inner, rdb: rdb, ttl: ttl}
}

func (c *CachingRepository) Init() error { return c.inner.Init() }

// Route methods are delegated without caching for now
func (c *CachingRepository) ListRoutes(ctx context.Context, serviceID string) ([]*Route, error) {
	return c.inner.ListRoutes(ctx, serviceID)
}
func (c *CachingRepository) GetRoute(ctx context.Context, serviceID, routeID string) (*Route, error) {
	return c.inner.GetRoute(ctx, serviceID, routeID)
}
func (c *CachingRepository) CreateRoute(ctx context.Context, r *Route) error {
	return c.inner.CreateRoute(ctx, r)
}
func (c *CachingRepository) UpdateRoute(ctx context.Context, r *Route) error {
	return c.inner.UpdateRoute(ctx, r)
}
func (c *CachingRepository) DeleteRoute(ctx context.Context, serviceID, routeID string) error {
	return c.inner.DeleteRoute(ctx, serviceID, routeID)
}
func (c *CachingRepository) FindRoute(ctx context.Context, serviceID, method, path string) (*Route, error) {
	return c.inner.FindRoute(ctx, serviceID, method, path)
}

func (c *CachingRepository) LoadEnabled(ctx context.Context) ([]*Service, error) {
	key := "gateway:services:enabled"
	if bs, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var list []*Service
		if json.Unmarshal(bs, &list) == nil {
			return list, nil
		}
	}
	list, err := c.inner.LoadEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if bs, err := json.Marshal(list); err == nil {
		_ = c.rdb.Set(ctx, key, bs, c.ttl).Err()
	}
	return list, nil
}

func (c *CachingRepository) List(ctx context.Context) ([]*Service, error) {
	key := "gateway:services:list"
	if bs, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var list []*Service
		if json.Unmarshal(bs, &list) == nil {
			return list, nil
		}
	}
	list, err := c.inner.List(ctx)
	if err != nil {
		return nil, err
	}
	if bs, err := json.Marshal(list); err == nil {
		_ = c.rdb.Set(ctx, key, bs, c.ttl).Err()
	}
	return list, nil
}

func (c *CachingRepository) Get(ctx context.Context, id string) (*Service, error) {
	key := "gateway:service:" + id
	if bs, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var s Service
		if json.Unmarshal(bs, &s) == nil {
			return &s, nil
		}
	}
	s, err := c.inner.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if bs, err := json.Marshal(s); err == nil {
		_ = c.rdb.Set(ctx, key, bs, c.ttl).Err()
	}
	return s, nil
}

func (c *CachingRepository) Create(ctx context.Context, s *Service) error {
	if err := c.inner.Create(ctx, s); err != nil {
		return err
	}
	c.invalidate(ctx, s.ID)
	return nil
}

func (c *CachingRepository) Update(ctx context.Context, s *Service) error {
	if err := c.inner.Update(ctx, s); err != nil {
		return err
	}
	c.invalidate(ctx, s.ID)
	return nil
}

func (c *CachingRepository) Delete(ctx context.Context, id string) error {
	if err := c.inner.Delete(ctx, id); err != nil {
		return err
	}
	c.invalidate(ctx, id)
	return nil
}

func (c *CachingRepository) invalidate(ctx context.Context, id string) {
	_ = c.rdb.Del(ctx, "gateway:services:enabled").Err()
	_ = c.rdb.Del(ctx, "gateway:services:list").Err()
	if id != "" {
		_ = c.rdb.Del(ctx, "gateway:service:"+id).Err()
	}
}
