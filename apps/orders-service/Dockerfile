# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN apk add --no-cache protobuf
RUN GOBIN=/usr/local/bin go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.1
RUN GOBIN=/usr/local/bin go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
COPY . .
RUN protoc --go_out=. --go-grpc_out=. proto/orders.proto
RUN go mod tidy
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /out/orders-service

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/orders-service /app/orders-service
USER 65532:65532
ENV PORT=8083
ENV GRPC_PORT=9093
EXPOSE 8083 9093
ENTRYPOINT ["/app/orders-service"]
