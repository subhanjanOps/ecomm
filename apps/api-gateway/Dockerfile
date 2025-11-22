# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN apk add --no-cache git
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN go mod tidy
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /out/api-gateway ./cmd/api-gateway

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/api-gateway /app/api-gateway
USER 65532:65532
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/api-gateway"]
