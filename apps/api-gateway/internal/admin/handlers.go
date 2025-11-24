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
	if body.PublicPrefix == "" {
		http.Error(w, "public_prefix required", http.StatusBadRequest)
		return
	}
	protocol := strings.ToLower(strings.TrimSpace(body.Protocol))
	if protocol == "" {
		protocol = "http"
	}
	var swJSON any
	base := strings.TrimSpace(body.BaseURL)
	if protocol == "http" {
		if body.SwaggerURL == "" {
			http.Error(w, "swagger_url required for protocol=http", http.StatusBadRequest)
			return
		}
		var inferredBase string
		var err error
		swJSON, inferredBase, err = fetchSwagger(r.Context(), body.SwaggerURL)
		if err != nil {
			http.Error(w, "failed to fetch swagger: "+err.Error(), http.StatusBadGateway)
			return
		}
		if base == "" {
			base = inferredBase
		}
		if base == "" {
			http.Error(w, "base_url missing and not derivable from swagger servers", http.StatusBadRequest)
			return
		}
	} else if protocol == "grpc-json" {
		if strings.TrimSpace(body.GRPCTarget) == "" {
			http.Error(w, "grpc_target required for protocol=grpc-json", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "unsupported protocol", http.StatusBadRequest)
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
		Protocol:      protocol,
		GRPCTarget:    strings.TrimSpace(body.GRPCTarget),
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
	if strings.ToLower(svc.Protocol) == "grpc-json" {
		http.Error(w, "refresh not supported for protocol=grpc-json", http.StatusBadRequest)
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
	// discover: /admin/services/{id}/routes/discover
	if strings.HasSuffix(id, "/routes/discover") && r.Method == http.MethodGet {
		base := strings.TrimSuffix(id, "/routes/discover")
		if strings.HasSuffix(base, "/") {
			base = strings.TrimSuffix(base, "/")
		}
		h.DiscoverRoutes(w, r, base)
		return
	}
	// bulk discover: /admin/services/{id}/routes/discover/bulk
	if strings.HasSuffix(id, "/routes/discover/bulk") && r.Method == http.MethodPost {
		base := strings.TrimSuffix(id, "/routes/discover/bulk")
		if strings.HasSuffix(base, "/") {
			base = strings.TrimSuffix(base, "/")
		}
		h.BulkAddDiscoveredRoutes(w, r, base)
		return
	}
	// refresh endpoint: /admin/services/{id}/refresh
	if strings.HasSuffix(id, "/refresh") && r.Method == http.MethodPost {
		id = strings.TrimSuffix(id, "/refresh")
		h.RefreshService(w, r, id)
		return
	}
	// routes collection: /admin/services/{id}/routes
	if strings.HasSuffix(id, "/routes") {
		id = strings.TrimSuffix(id, "/routes")
		h.Routes(w, r, id)
		return
	}
	// routes detail: /admin/services/{id}/routes/{rid}
	if strings.Contains(id, "/routes/") {
		parts := strings.SplitN(id, "/routes/", 2)
		if len(parts) == 2 {
			serviceID := parts[0]
			routeID := parts[1]
			h.RouteByID(w, r, serviceID, routeID)
			return
		}
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

// Routes lists or creates route mappings for a service
func (h *Handler) Routes(w http.ResponseWriter, r *http.Request, serviceID string) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.repo.ListRoutes(r.Context(), serviceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		util.JSON(w, list)
	case http.MethodPost:
		var body struct {
			Method       string                     `json:"method"`
			Path         string                     `json:"path"`
			GRPCMethod   string                     `json:"grpc_method"`
			QueryMapping registry.RouteQueryMapping `json:"query_mapping"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Method == "" || body.Path == "" || body.GRPCMethod == "" {
			http.Error(w, "method, path, grpc_method required", http.StatusBadRequest)
			return
		}
		rt := &registry.Route{ID: uuid.NewString(), ServiceID: serviceID, Method: body.Method, Path: body.Path, GRPCMethod: body.GRPCMethod, QueryMapping: body.QueryMapping, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if err := h.repo.CreateRoute(r.Context(), rt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		util.JSON(w, rt)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// RouteByID retrieves/updates/deletes a specific route
func (h *Handler) RouteByID(w http.ResponseWriter, r *http.Request, serviceID, routeID string) {
	switch r.Method {
	case http.MethodGet:
		rt, err := h.repo.GetRoute(r.Context(), serviceID, routeID)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		util.JSON(w, rt)
	case http.MethodPut, http.MethodPatch:
		var body registry.Route
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body.ID = routeID
		body.ServiceID = serviceID
		body.UpdatedAt = time.Now()
		if err := h.repo.UpdateRoute(r.Context(), &body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		util.JSON(w, body)
	case http.MethodDelete:
		if err := h.repo.DeleteRoute(r.Context(), serviceID, routeID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		util.JSON(w, map[string]any{"deleted": routeID})
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// DiscoverRoutes lists gRPC methods available on a service's grpc_target using reflection.
// Returns an array of { service, method, grpc_method }.
func (h *Handler) DiscoverRoutes(w http.ResponseWriter, r *http.Request, serviceID string) {
	svc, err := h.repo.Get(r.Context(), serviceID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if strings.ToLower(svc.Protocol) != "grpc-json" || svc.GRPCTarget == "" {
		http.Error(w, "service is not grpc-json or grpc_target missing", http.StatusBadRequest)
		return
	}
	list, err := discoverGRPCMethods(r.Context(), svc.GRPCTarget)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	util.JSON(w, list)
}

// BulkAddDiscoveredRoutes creates REST routes for all discovered gRPC methods using a default strategy.
// Heuristic: GET for List/Get*, POST for Create*, PUT for Update*, DELETE for Delete*, otherwise GET.
// Path defaults to kebab-case of method name: /list-users, /get-user, etc.
func (h *Handler) BulkAddDiscoveredRoutes(w http.ResponseWriter, r *http.Request, serviceID string) {
	svc, err := h.repo.Get(r.Context(), serviceID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if strings.ToLower(svc.Protocol) != "grpc-json" || svc.GRPCTarget == "" {
		http.Error(w, "service is not grpc-json or grpc_target missing", http.StatusBadRequest)
		return
	}
	list, err := discoverGRPCMethods(r.Context(), svc.GRPCTarget)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	// Existing routes to avoid duplicates
	existing, _ := h.repo.ListRoutes(r.Context(), serviceID)
	exists := map[string]bool{}
	for _, rt := range existing {
		key := strings.ToUpper(rt.Method) + " " + rt.Path
		exists[key] = true
	}
	created := 0
	for _, d := range list {
		method := "GET"
		upper := strings.ToUpper(d.Method)
		switch {
		case strings.HasPrefix(upper, "CREATE"):
			method = "POST"
		case strings.HasPrefix(upper, "UPDATE"):
			method = "PUT"
		case strings.HasPrefix(upper, "DELETE"):
			method = "DELETE"
		default:
			method = "GET"
		}
		path := "/" + toKebab(d.Method)
		key := method + " " + path
		if exists[key] {
			continue
		}
		rt := &registry.Route{ID: uuid.NewString(), ServiceID: serviceID, Method: method, Path: path, GRPCMethod: d.GRPCMethod, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if err := h.repo.CreateRoute(r.Context(), rt); err == nil {
			created++
		}
	}
	util.JSON(w, map[string]any{"created": created})
}

func toKebab(s string) string {
	out := make([]rune, 0, len(s)*2)
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '-')
		}
		out = append(out, rune(strings.ToLower(string(r))[0]))
	}
	return string(out)
}
