# =============================================================================
# Etapa 1: Builder
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25.0-alpine3.21 AS builder

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Compilación con cross-compilation:
# 1. magi-api: Microservicio de consultas HTTP (CQRS Read Path)
# 2. magi-worker: Microservicio ingestor de Kafka (CQRS Write Path)
# 3. magi-service: Binario unificado conjunto (cmd/all/main.go)
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /magi-api \
      ./cmd/server/main.go

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /magi-worker \
      ./cmd/worker/main.go

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /magi-service \
      ./cmd/all/main.go

# =============================================================================
# Etapa 2: Runtime
# =============================================================================
FROM alpine:3.21

LABEL org.opencontainers.image.title="AIRES MAGI Backend" \
      org.opencontainers.image.description="Microservicios CQRS de ingesta Kafka (worker) y consultas HTTP (api)" \
      org.opencontainers.image.source="https://github.com/YESUBZERO/MAGI-backend"

RUN apk add --no-cache ca-certificates tzdata procps && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /magi-api /usr/local/bin/magi-api
COPY --from=builder --chown=appuser:appgroup /magi-worker /usr/local/bin/magi-worker
COPY --from=builder --chown=appuser:appgroup /magi-service /usr/local/bin/magi-service

COPY --chown=appuser:appgroup entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD pgrep -x magi-worker > /dev/null || wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["api"]
