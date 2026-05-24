# Imagen base
FROM --platform=$BUILDPLATFORM golang:1.23.5-alpine3.21 as builder 

# Argumentos que Docker llea automaticamente
ARG TARGETOS
ARG TARGETARCH

# Instalamos dependencias necesarias
RUN apk add --no-cache git

# Directorio de trabajo
WORKDIR /app

# Copiar dependencias y codigo fuente
COPY go.mod go.sum ./
RUN go mod download

# Copiar el codigo fuente
COPY . .

# Compilar el binario
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /consumer-service ./cmd/main.go


# Etapa 2: Ejectuar el binario
FROM alpine:latest

# Crear un usuario y grupo
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Directorio de trabajo
WORKDIR /home/appuser/

# Copiar el binario de la etapa anterior
COPY --from=builder /consumer-service .

# Cambiar el propietario del binario
RUN chown appuser:appgroup consumer-service

# Cambiar al usuario
USER appuser

# Ejecutar el binario
CMD ["./consumer-service"]