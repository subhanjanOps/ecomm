package registry

import "time"

// RouteQueryMapEntry maps a query param to an RPC field with a type hint.
// Type can be: string (default), int, float, bool.
type RouteQueryMapEntry struct {
	Field string `json:"field"`
	Type  string `json:"type"`
}

type RouteQueryMapping map[string]RouteQueryMapEntry

// Route maps an incoming REST method+path (under a service's public prefix)
// to a gRPC full method name (package.Service/Method) for transcoding.
// Path can contain template params like {id} or with type hints {id:int}.
// QueryMapping optionally maps query parameters to RPC fields with type hints.
type Route struct {
	ID           string            `json:"id"`
	ServiceID    string            `json:"service_id"`
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	GRPCMethod   string            `json:"grpc_method"`
	QueryMapping RouteQueryMapping `json:"query_mapping,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}
