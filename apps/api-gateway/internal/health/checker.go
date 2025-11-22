package health

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"ecomm/api-gateway/internal/registry"
)

// Start launches a background health checker that updates service status periodically.
// intervalSec is a string integer number of seconds.
func Start(repo registry.Repository, intervalSec string) {
	sec, _ := strconv.Atoi(intervalSec)
	if sec <= 0 {
		sec = 30
	}
	t := time.NewTicker(time.Duration(sec) * time.Second)
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		for range t.C {
			list, err := repo.List(context.Background())
			if err != nil {
				continue
			}
			for _, s := range list {
				if !s.Enabled || s.BaseURL == "" {
					continue
				}
				// probe /healthz
				url := s.BaseURL
				if len(url) > 0 && url[len(url)-1] == '/' {
					url = url[:len(url)-1]
				}
				req, _ := http.NewRequest(http.MethodGet, url+"/healthz", nil)
				resp, err := client.Do(req)
				status := "Unhealthy"
				if err == nil && resp.StatusCode/100 == 2 {
					status = "Healthy"
				}
				// update fields
				s.LastStatus = status
				s.LastHealthAt = time.Now()
				_ = repo.Update(context.Background(), s)
			}
		}
	}()
}
