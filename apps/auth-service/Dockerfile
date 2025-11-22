# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /out/auth-service

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/auth-service /app/auth-service
USER 65532:65532
ENV PORT=8084
EXPOSE 8084
ENTRYPOINT ["/app/auth-service"]
