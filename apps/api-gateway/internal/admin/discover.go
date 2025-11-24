package admin

import (
	"context"
	"sort"

	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

type discoveredMethod struct {
	Service    string `json:"service"`
	Method     string `json:"method"`
	GRPCMethod string `json:"grpc_method"`
}

func discoverGRPCMethods(ctx context.Context, target string) ([]discoveredMethod, error) {
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rc := grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))
	defer rc.Reset()

	svcs, err := rc.ListServices()
	if err != nil {
		return nil, err
	}
	sort.Strings(svcs)
	var out []discoveredMethod
	for _, s := range svcs {
		// skip reflection service itself
		if s == "grpc.reflection.v1alpha.ServerReflection" {
			continue
		}
		desc, err := rc.ResolveService(s)
		if err != nil {
			continue
		}
		for _, m := range desc.GetMethods() {
			out = append(out, discoveredMethod{Service: s, Method: m.GetName(), GRPCMethod: s + "/" + m.GetName()})
		}
	}
	return out, nil
}
