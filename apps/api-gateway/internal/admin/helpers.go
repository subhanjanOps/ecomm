package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
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
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
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
