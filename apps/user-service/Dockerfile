# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN apk add --no-cache protobuf
# Install protoc plugins
RUN GOBIN=/usr/local/bin go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.1
RUN GOBIN=/usr/local/bin go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
COPY . .
# Generate protobuf code
RUN protoc --go_out=. --go-grpc_out=. proto/user.proto
# Ensure module deps and sums are updated
RUN go mod tidy
# Build binary
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /out/user-service

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/user-service /app/user-service
USER 65532:65532
ENV PORT=8081
ENV GRPC_PORT=9091
EXPOSE 8081 9091
ENTRYPOINT ["/app/user-service"]
