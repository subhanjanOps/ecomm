package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func fetchSwagger(ctx context.Context, urlStr string) (any, string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, "", &statusErr{code: resp.StatusCode}
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	// Validate OpenAPI
	loader := &openapi3.Loader{IsExternalRefsAllowed: true}
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, "", err
	}
	if err := doc.Validate(ctx); err != nil {
		return nil, "", err
	}
	// decode to generic map for storage
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, "", err
	}
	base := ""
	if v, ok := m["servers"].([]any); ok && len(v) > 0 {
		if first, ok := v[0].(map[string]any); ok {
			if u, ok := first["url"].(string); ok {
				base = strings.TrimRight(u, "/")
			}
		}
	}
	_ = doc // currently unused besides validation
	return m, base, nil
}

type statusErr struct{ code int }

func (e *statusErr) Error() string { return "http status" }

func normalizePrefix(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasSuffix(p, "/") {
		p = p + "/"
	}
	return p
}

func guessNameFromURL(u string) string {
	if strings.HasPrefix(u, "http") {
		if parsed, err := url.Parse(u); err == nil {
			host := parsed.Hostname()
			if host != "" {
				return host
			}
		}
	}
	return u
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
