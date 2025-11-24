package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"ecomm/api-gateway/internal/grpcjson"
	"ecomm/api-gateway/internal/registry"
)

// Dynamic returns an http.HandlerFunc that proxies requests based on the registry
func Dynamic(reg *registry.Registry, repo registry.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, remainder, ok := reg.Match(r.URL.Path)
		if !ok || svc == nil || !svc.Enabled {
			http.NotFound(w, r)
			return
		}
		// If service requests HTTP→gRPC transcoding, route via JSON transcoder
		if strings.ToLower(svc.Protocol) == "grpc-json" {
			if svc.GRPCTarget == "" {
				http.Error(w, "grpc target missing", http.StatusBadGateway)
				return
			}
			methodPath := strings.TrimPrefix(remainder, "/")
			params := map[string]any{}
			// If remainder isn't a direct gRPC method, consult route mappings with templating
			if !strings.Contains(methodPath, "/") || !strings.Contains(methodPath, ".") {
				if repo != nil {
					if routes, err := repo.ListRoutes(r.Context(), svc.ID); err == nil {
						if rt, pm := matchTemplatedRoute(routes, r.Method, remainder); rt != nil {
							methodPath = strings.TrimPrefix(rt.GRPCMethod, "/")
							params = pm
							mergeQueryParams(params, r.URL, rt)
						}
					}
				}
			}
			grpcjson.ServeWithParams(svc.GRPCTarget, methodPath, params, w, r)
			return
		}

		target, err := url.Parse(svc.BaseURL)
		if err != nil {
			http.Error(w, "bad upstream", http.StatusBadGateway)
			return
		}
		director := func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			basePath := strings.TrimSuffix(target.Path, "/")
			upPath := basePath + remainder
			req.URL.Path = upPath
			req.URL.RawPath = upPath
			req.Host = target.Host
		}
		rp := &httputil.ReverseProxy{Director: director}
		rp.ServeHTTP(w, r)
	}
}

// matchTemplatedRoute tries to match method+path against a set of routes supporting
// simple templates like /users/{id}. Returns the matched route and extracted params.
func matchTemplatedRoute(routes []*registry.Route, method, path string) (*registry.Route, map[string]any) {
	method = strings.ToUpper(method)
	// Prefer longer patterns (more specific)
	var best *registry.Route
	var bestParams map[string]any
	for _, rt := range routes {
		if strings.ToUpper(rt.Method) != method {
			continue
		}
		if pm, ok := matchPattern(rt.Path, path); ok {
			if best == nil || len(rt.Path) > len(best.Path) {
				best = rt
				bestParams = pm
			}
		}
	}
	return best, bestParams
}

// matchPattern checks if request path matches a template pattern and extracts params.
// Pattern and path are absolute paths starting with '/'. Template segments like {name} match
// a single path segment and are returned in params.
func matchPattern(pattern, path string) (map[string]any, bool) {
	p := strings.TrimSuffix(pattern, "/")
	u := strings.TrimSuffix(path, "/")
	if p == "" {
		p = "/"
	}
	if u == "" {
		u = "/"
	}
	ps := strings.Split(strings.TrimPrefix(p, "/"), "/")
	us := strings.Split(strings.TrimPrefix(u, "/"), "/")
	if len(ps) != len(us) {
		return nil, false
	}
	params := map[string]any{}
	for i := 0; i < len(ps); i++ {
		segP := ps[i]
		segU := us[i]
		if len(segP) >= 2 && segP[0] == '{' && segP[len(segP)-1] == '}' {
			spec := segP[1 : len(segP)-1]
			key := spec
			typ := "string"
			if i := strings.Index(spec, ":"); i > 0 {
				key = spec[:i]
				typ = spec[i+1:]
			}
			if key == "" {
				return nil, false
			}
			params[key] = coerceType(segU, typ)
			continue
		}
		if segP != segU {
			return nil, false
		}
	}
	return params, true
}

// mergeQueryParams maps query values to rpc fields using route.QueryMapping with type coercion
func mergeQueryParams(params map[string]any, u *url.URL, rt *registry.Route) {
	if rt == nil || rt.QueryMapping == nil || u == nil {
		return
	}
	q := u.Query()
	for qp, entry := range rt.QueryMapping {
		if vals, ok := q[qp]; ok && len(vals) > 0 {
			v := vals[0]
			if entry.Field == "" {
				continue
			}
			params[entry.Field] = coerceType(v, entry.Type)
		}
	}
}

func coerceType(v string, typ string) any {
	switch strings.ToLower(typ) {
	case "int", "integer":
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
		return v
	case "float", "double", "number":
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
		return v
	case "bool", "boolean":
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
		return v
	default:
		return v
	}
}
