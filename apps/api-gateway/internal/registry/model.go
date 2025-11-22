package registry

import "time"

// Service represents a backend service managed by the gateway
type Service struct {
	ID            string    `json:"id" example:"3d1a7e94-0a2f-4a49-9a9b-8f9f2d0c6f67"`
	Name          string    `json:"name" example:"User Service"`
	Description   string    `json:"description,omitempty" example:"Manages users"`
	PublicPrefix  string    `json:"public_prefix" example:"/api/users/"`
	BaseURL       string    `json:"base_url" example:"http://user-service:8081"`
	SwaggerURL    string    `json:"swagger_url" example:"http://user-service:8081/swagger.json"`
	Enabled       bool      `json:"enabled" example:"true"`
	SwaggerJSON   any       `json:"swagger_json,omitempty"`
	LastRefreshed time.Time `json:"last_refreshed_at,omitempty" example:"2025-11-22T10:20:30Z"`
	LastHealthAt  time.Time `json:"last_health_at,omitempty" example:"2025-11-22T10:20:00Z"`
	LastStatus    string    `json:"last_status,omitempty" example:"Healthy"`
	CreatedAt     time.Time `json:"created_at" example:"2025-11-22T10:00:00Z"`
	UpdatedAt     time.Time `json:"updated_at" example:"2025-11-22T10:10:00Z"`
}
