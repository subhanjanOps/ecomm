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

// Services handles list and create
// @Summary List or create services
// @Tags admin
// @Produce json
// @Success 200 {array} registry.Service
// @Router /admin/services [get]
//
// @Summary Create service
// @Tags admin
// @Accept json
// @Produce json
// @Param payload body admin.CreateServiceRequest true "Service payload"
// @Success 200 {object} registry.Service
// @Failure 400 {string} string "bad request"
// @Failure 401 {string} string "unauthorized"
// @Failure 500 {string} string "internal error"
// @Security BearerAuth
// @Router /admin/services [post]
func (h *Handler) Services(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
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
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// ServiceByID handles get/put/delete and refresh action
// @Summary Get service by ID
// @Tags admin
// @Produce json
// @Param id path string true "Service ID"
// @Success 200 {object} registry.Service
// @Failure 404
// @Failure 401 {string} string
// @Security BearerAuth
// @Router /admin/services/{id} [get]
//
// @Summary Update service
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Service ID"
// @Param payload body registry.Service true "Service"
// @Success 200 {object} registry.Service
// @Failure 400 {string} string
// @Failure 401 {string} string
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /admin/services/{id} [put]
//
// @Summary Delete service
// @Tags admin
// @Param id path string true "Service ID"
// @Success 200 {object} map[string]string
// @Failure 401 {string} string
// @Security BearerAuth
// @Router /admin/services/{id} [delete]
//
// @Summary Refresh service swagger
// @Tags admin
// @Param id path string true "Service ID"
// @Success 200 {object} registry.Service
// @Failure 401 {string} string
// @Failure 502 {string} string "bad gateway"
// @Security BearerAuth
// @Router /admin/services/{id}/refresh [post]
func (h *Handler) ServiceByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/services/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(id, "/refresh") && r.Method == http.MethodPost {
		id = strings.TrimSuffix(id, "/refresh")
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
		return
	}
	switch r.Method {
	case http.MethodGet:
		svc, err := h.repo.Get(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		util.JSON(w, svc)
	case http.MethodPut:
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
	case http.MethodDelete:
		if err := h.repo.Delete(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = registry.LoadEnabled(h.repo, h.reg)
		util.JSON(w, map[string]string{"deleted": id})
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}
