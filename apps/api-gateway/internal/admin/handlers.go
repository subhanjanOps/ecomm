package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"ecomm/api-gateway/internal/registry"
	"ecomm/api-gateway/internal/util"
)

type Handler struct {
	repo registry.Repository
	reg  *registry.Registry
}

func NewHandler(repo registry.Repository, reg *registry.Registry) *Handler {
	return &Handler{repo: repo, reg: reg}
}

// ListServices returns all registered services.
// @Summary List services
// @Tags admin
// @Produce json
// @Success 200 {array} registry.Service
// @Router /admin/services [get]
func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	list, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []*registry.Service{}
	}
	util.JSON(w, list)
}

// CreateService onboards a new backend service.
// @Summary Create service
// @Tags admin
// @Accept json
// @Param payload body admin.CreateServiceRequest true "Service payload"
// @Success 200 {object} registry.Service
// @Failure 400 {string} string "bad request"
// @Failure 401 {string} string "unauthorized"
// @Failure 500 {string} string "internal error"
// @Security BearerAuth
// @Router /admin/services [post]
func (h *Handler) CreateService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.PublicPrefix == "" || body.SwaggerURL == "" {
		http.Error(w, "public_prefix and swagger_url required", http.StatusBadRequest)
		return
	}
	swJSON, inferredBase, err := fetchSwagger(r.Context(), body.SwaggerURL)
	if err != nil {
		http.Error(w, "failed to fetch swagger: "+err.Error(), http.StatusBadGateway)
		return
	}
	base := body.BaseURL
	if base == "" {
		base = inferredBase
	}
	if base == "" {
		http.Error(w, "base_url missing and not derivable from swagger servers", http.StatusBadRequest)
		return
	}
	en := true
	if body.Enabled != nil {
		en = *body.Enabled
	}
	svc := &registry.Service{
		ID:            uuid.NewString(),
		Name:          firstNonEmpty(body.Name, guessNameFromURL(base)),
		Description:   body.Description,
		PublicPrefix:  normalizePrefix(body.PublicPrefix),
		BaseURL:       strings.TrimRight(base, "/"),
		SwaggerURL:    body.SwaggerURL,
		Enabled:       en,
		SwaggerJSON:   swJSON,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		LastRefreshed: time.Now(),
	}
	if err := h.repo.Create(r.Context(), svc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = registry.LoadEnabled(h.repo, h.reg)
	util.JSON(w, svc)
}

// Services is a thin dispatcher that routes to method-specific handlers (keeps existing route).
func (h *Handler) Services(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListServices(w, r)
	case http.MethodPost:
		h.CreateService(w, r)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// GetService retrieves a service by ID.
// @Summary Get service by ID
// @Tags admin
// @Produce json
// @Param id path string true "Service ID"
// @Success 200 {object} registry.Service
// @Failure 404
// @Failure 401 {string} string
// @Security BearerAuth
// @Router /admin/services/{id} [get]
func (h *Handler) GetService(w http.ResponseWriter, r *http.Request, id string) {
	svc, err := h.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	util.JSON(w, svc)
}

// UpdateService updates an existing service.
// @Summary Update service
// @Tags admin
// @Accept json
// @Param id path string true "Service ID"
// @Param payload body registry.Service true "Service"
// @Success 200 {object} registry.Service
// @Failure 400 {string} string
// @Failure 401 {string} string
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /admin/services/{id} [put]
func (h *Handler) UpdateService(w http.ResponseWriter, r *http.Request, id string) {
	var body registry.Service
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body.ID = id
	if err := h.repo.Update(r.Context(), &body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = registry.LoadEnabled(h.repo, h.reg)
	util.JSON(w, body)
}

// DeleteService removes a service registration.
// @Summary Delete service
// @Tags admin
// @Param id path string true "Service ID"
// @Success 200 {object} map[string]string
// @Failure 401 {string} string
// @Security BearerAuth
// @Router /admin/services/{id} [delete]
func (h *Handler) DeleteService(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = registry.LoadEnabled(h.repo, h.reg)
	util.JSON(w, map[string]string{"deleted": id})
}

// RefreshService re-fetches and validates the service swagger then updates the record.
// @Summary Refresh service swagger
// @Tags admin
// @Param id path string true "Service ID"
// @Success 200 {object} registry.Service
// @Failure 401 {string} string
// @Failure 502 {string} string "bad gateway"
// @Security BearerAuth
// @Router /admin/services/{id}/refresh [post]
func (h *Handler) RefreshService(w http.ResponseWriter, r *http.Request, id string) {
	svc, err := h.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	swJSON, inferredBase, err := fetchSwagger(r.Context(), svc.SwaggerURL)
	if err != nil {
		http.Error(w, "failed to fetch swagger: "+err.Error(), http.StatusBadGateway)
		return
	}
	if svc.BaseURL == "" && inferredBase != "" {
		svc.BaseURL = inferredBase
	}
	svc.SwaggerJSON = swJSON
	svc.LastRefreshed = time.Now()
	if err := h.repo.Update(r.Context(), svc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = registry.LoadEnabled(h.repo, h.reg)
	util.JSON(w, svc)
}

// ServiceByID dispatches path-based actions to their method-specific handlers.
func (h *Handler) ServiceByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/services/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	// refresh endpoint: /admin/services/{id}/refresh
	if strings.HasSuffix(id, "/refresh") && r.Method == http.MethodPost {
		id = strings.TrimSuffix(id, "/refresh")
		h.RefreshService(w, r, id)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.GetService(w, r, id)
	case http.MethodPut:
		h.UpdateService(w, r, id)
	case http.MethodDelete:
		h.DeleteService(w, r, id)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}
