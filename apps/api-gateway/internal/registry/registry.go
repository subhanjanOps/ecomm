package registry

import (
	"sort"
	"strings"
	"sync"
)

// Registry holds enabled services and performs prefix matching
type Registry struct {
	mu       sync.RWMutex
	byPrefix map[string]*Service
	order    []string // prefixes sorted by length desc
}

func New() *Registry {
	return &Registry{byPrefix: map[string]*Service{}}
}

// Set replaces the current registry content with provided services (enabled ones only)
func (r *Registry) Set(services []*Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byPrefix = map[string]*Service{}
	for _, s := range services {
		if s.Enabled {
			r.byPrefix[s.PublicPrefix] = s
		}
	}
	r.order = make([]string, 0, len(r.byPrefix))
	for p := range r.byPrefix {
		r.order = append(r.order, p)
	}
	sort.Slice(r.order, func(i, j int) bool { return len(r.order[i]) > len(r.order[j]) })
}

// Match finds the service by longest matching prefix and returns remainder path
func (r *Registry) Match(path string) (*Service, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.order {
		if strings.HasPrefix(path, p) {
			remainder := strings.TrimPrefix(path, p)
			if !strings.HasPrefix(remainder, "/") {
				remainder = "/" + remainder
			}
			return r.byPrefix[p], remainder, true
		}
	}
	return nil, "", false
}
