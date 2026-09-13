# =============================================================================
# Etapa 1: Builder
# --platform=$BUILDPLATFORM usa la arquitectura del host para compilar,
# mientras que TARGETOS/TARGETARCH definen el binario final (cross-compile).
# Soporta: linux/amd64 y linux/arm64
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25.0-alpine3.21 AS builder

# Argumentos inyectados automáticamente por Docker BuildKit
ARG TARGETOS
ARG TARGETARCH

# Instalar dependencias del sistema necesarias para compilación
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copiar manifiestos de módulos primero para aprovechar la caché de capas
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copiar el código fuente completo
COPY . .

# Compilar los binarios con cross-compilation habilitada:
# 1. magi-api: Microservicio de consultas HTTP (CQRS Read Path)
# 2. magi-worker: Microservicio ingestor de Kafka (CQRS Write Path)
# 3. magi-service: Binario conjunto unificado (cmd/main.go para desarrollo)
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /magi-api \
      ./cmd/api/main.go

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
      ./cmd/main.go

# =============================================================================
# Etapa 2: Imagen de ejecución (runtime unificado con soporte CQRS)
# =============================================================================
FROM alpine:3.21

# Metadatos OCI para trazabilidad en registros de contenedores
LABEL org.opencontainers.image.title="MAGI Backend Service" \
      org.opencontainers.image.description="Microservicios CQRS de ingesta Kafka (worker) y consultas HTTP (api) para el proyecto MAGI" \
      org.opencontainers.image.source="https://github.com/YESUBZERO/MAGI-backend" \
      org.opencontainers.image.licenses="MIT"

# Certificados CA, zona horaria y procps para monitoreo de procesos
RUN apk add --no-cache ca-certificates tzdata procps && \
    # Crear usuario y grupo sin privilegios
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

# Directorio de trabajo seguro e independiente
WORKDIR /app

# Copiar los binarios compilados desde el builder directamente al path del sistema
COPY --from=builder --chown=appuser:appgroup /magi-api /usr/local/bin/magi-api
COPY --from=builder --chown=appuser:appgroup /magi-worker /usr/local/bin/magi-worker
COPY --from=builder --chown=appuser:appgroup /magi-service /usr/local/bin/magi-service

# Copiar y asegurar permisos del script de entrada (dispatcher CQRS)
COPY --chown=appuser:appgroup scripts/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Cambiar al usuario sin privilegios
USER appuser

# Puerto expuesto por el servidor HTTP de Gin (magi-api)
EXPOSE 8080

# Health check adaptativo:
# - Si corre magi-worker: valida que el proceso esté vivo en segundo plano.
# - Si corre magi-api o magi-service: valida que el endpoint /health responda HTTP 200.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD pgrep -x magi-worker > /dev/null || wget -qO- http://localhost:8080/health || exit 1

# Punto de entrada despachador: acepta 'api', 'worker', 'monolith' o comandos directos
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["api"]
