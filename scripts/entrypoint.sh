#!/bin/sh
set -e

# =============================================================================
# MAGI Backend - Entrypoint Dispatcher (CQRS)
# Permite ejecutar el rol deseado según el argumento proporcionado:
# - 'api'      o 'magi-api'     -> Microservicio de Consultas HTTP (Read Path)
# - 'worker'   o 'magi-worker'  -> Microservicio Ingestor de Kafka (Write Path)
# - 'monolith' o 'magi-service' -> Ejecución conjunta tradicional (cmd/main.go)
# - Cualquier otro comando      -> Ejecución directa (sh, bash, etc.)
# =============================================================================

case "$1" in
  api|magi-api)
    shift
    exec /usr/local/bin/magi-api "$@"
    ;;
  worker|magi-worker)
    shift
    exec /usr/local/bin/magi-worker "$@"
    ;;
  monolith|magi-service)
    shift
    exec /usr/local/bin/magi-service "$@"
    ;;
  *)
    exec "$@"
    ;;
esac
