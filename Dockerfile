# =============================================================================
# Etapa 1: Builder
# --platform=$BUILDPLATFORM usa la arquitectura del host para compilar,
# mientras que TARGETOS/TARGETARCH definen el binario final (cross-compile).
# Soporta: linux/amd64 y linux/arm64
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.23.5-alpine3.21 AS builder

# Argumentos inyectados automáticamente por Docker BuildKit
ARG TARGETOS
ARG TARGETARCH

# Instalar dependencias del sistema necesarias para compilación
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copiar manifiestos de módulos primero para aprovechar la caché de capas.
# Las dependencias solo se re-descargan si go.mod o go.sum cambian.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copiar el código fuente
COPY . .

# Compilar el binario con cross-compilation habilitada.
# CGO_ENABLED=0  → binario estático sin dependencias de libc (compatible con scratch/alpine).
# -trimpath     → elimina rutas absolutas del binario (reproducibilidad).
# -ldflags      → reduce el tamaño eliminando información de debug y el símbolo de tabla.
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /magi-service \
      ./cmd/main.go

# =============================================================================
# Etapa 2: Imagen de ejecución (runtime)
# Se fija la versión de Alpine para builds reproducibles y deterministas.
# =============================================================================
FROM alpine:3.21

# Metadatos OCI para trazabilidad en registros de contenedores
LABEL org.opencontainers.image.title="MAGI Service" \
      org.opencontainers.image.description="Servicio consumidor de Kafka con API HTTP para el proyecto MAGI" \
      org.opencontainers.image.source="https://github.com/YESUBZERO/MAGI-backend" \
      org.opencontainers.image.licenses="MIT"

# Certificados CA y zona horaria (necesarios para conexiones TLS y logs correctos)
RUN apk add --no-cache ca-certificates tzdata && \
    # Crear usuario y grupo sin privilegios
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

# Directorio de trabajo seguro e independiente
WORKDIR /app

# Copiar el binario compilado desde el builder directamente al path del sistema
COPY --from=builder --chown=appuser:appgroup /magi-service /usr/local/bin/magi-service

# Cambiar al usuario sin privilegios
USER appuser

# Puerto expuesto por el servidor HTTP de Gin
EXPOSE 8080

# Health check: valida que el servidor HTTP responda correctamente.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# Punto de entrada del servicio usando la ruta absoluta
ENTRYPOINT ["/usr/local/bin/magi-service"]