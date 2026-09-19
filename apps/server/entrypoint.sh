#!/bin/sh
set -e

# =============================================================================
# MAGI Backend - Entrypoint Dispatcher (CQRS)
# =============================================================================

TARGET="$1"
if [ -n "$SERVICE_ROLE" ]; then
  TARGET="$SERVICE_ROLE"
fi

case "$1" in
  api|magi-api|worker|magi-worker|monolith|magi-service)
    shift
    ;;
esac

case "$TARGET" in
  api|magi-api)
    exec /usr/local/bin/magi-api "$@"
    ;;
  worker|magi-worker)
    exec /usr/local/bin/magi-worker "$@"
    ;;
  monolith|magi-service)
    exec /usr/local/bin/magi-service "$@"
    ;;
  *)
    exec "$@"
    ;;
esac
