package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"ecomm/api-gateway/internal/registry"
)

// Dynamic returns an http.HandlerFunc that proxies requests based on the registry
func Dynamic(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, remainder, ok := reg.Match(r.URL.Path)
		if !ok || svc == nil || !svc.Enabled {
			http.NotFound(w, r)
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
