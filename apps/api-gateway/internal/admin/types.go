package admin

// CreateServiceRequest is the request payload to create/register a service
type CreateServiceRequest struct {
	Name         string `json:"name" example:"User Service"`
	Description  string `json:"description" example:"Manages users"`
	PublicPrefix string `json:"public_prefix" example:"/api/users/"`
	SwaggerURL   string `json:"swagger_url" example:"http://user-service:8081/swagger.json"`
	BaseURL      string `json:"base_url" example:"http://user-service:8081"`
	Enabled      *bool  `json:"enabled" example:"true"`
}
