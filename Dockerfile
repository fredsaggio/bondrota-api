ARG GO_VERSION=1.26.0

FROM docker.io/library/golang:${GO_VERSION}-trixie AS builder
ARG GIT_COMMIT="dev"
ARG BUILD_TIME=""
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    LDFLAGS="-s -w -X github.com/fredsaggio/bondrota-api/internal/version.Commit=${GIT_COMMIT} -X github.com/fredsaggio/bondrota-api/internal/version.BuildTime=${BUILD_TIME}" && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="$LDFLAGS" -o /app/bin/api ./cmd/main.go ./cmd/dependencies.go

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /
COPY --from=builder /app/bin/api /api
COPY --from=builder /app/internal/db/migrations /db/migrations
EXPOSE 8080
ENTRYPOINT ["/api"]