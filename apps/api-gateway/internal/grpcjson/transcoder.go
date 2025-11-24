package grpcjson

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// Serve performs a minimal JSON→gRPC transcoding for unary RPCs using server reflection.
// Path remainder must be "/<package>.<Service>/<Method>". Request body should be JSON for the method's input message.
// NOTE: This is a minimal dynamic approach for exploration; production use should adopt google.api.http annotations
// and/or generated grpc-gateway handlers for robust REST shapes.
func Serve(grpcTarget, methodPath string, w http.ResponseWriter, r *http.Request) {
	full := strings.TrimPrefix(methodPath, "/")
	if full == "" || !strings.Contains(full, "/") {
		http.Error(w, "invalid gRPC method path; expected /package.Service/Method", http.StatusBadRequest)
		return
	}
	// Dial upstream
	conn, err := grpc.DialContext(r.Context(), grpcTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		http.Error(w, "upstream dial failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer conn.Close()

	// Setup reflection client
	rc := grpcreflect.NewClient(r.Context(), reflectpb.NewServerReflectionClient(conn))
	defer rc.Reset()

	service := full[:strings.LastIndex(full, "/")]
	method := full[strings.LastIndex(full, "/")+1:]
	desc, err := rc.ResolveService(service)
	if err != nil {
		http.Error(w, "service not found: "+err.Error(), http.StatusBadRequest)
		return
	}
	md := desc.FindMethodByName(method)
	if md == nil {
		http.Error(w, "method not found", http.StatusBadRequest)
		return
	}
	// Build dynamic input message from JSON body
	inMsg := dynamic.NewMessage(md.GetInputType())
	body, _ := io.ReadAll(r.Body)
	if len(body) == 0 {
		body = []byte("{}")
	}
	if err := inMsg.UnmarshalJSON(body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Propagate Authorization header as metadata if present
	ctx := r.Context()
	if auth := r.Header.Get("Authorization"); auth != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", auth)
	}
	// Invoke unary RPC
	outMsg := dynamic.NewMessage(md.GetOutputType())
	err = conn.Invoke(ctx, "/"+service+"/"+method, inMsg, outMsg)
	if err != nil {
		http.Error(w, fmt.Sprintf("grpc error: %v", err), http.StatusBadGateway)
		return
	}
	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	bs, err := outMsg.MarshalJSON()
	if err != nil {
		http.Error(w, "marshal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(bs)
}
