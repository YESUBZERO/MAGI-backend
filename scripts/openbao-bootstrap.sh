#!/usr/bin/env bash
set -euo pipefail

KUBECONFIG_PATH="${KUBECONFIG:-$HOME/.kube/config-k3s-tailscale}"
OPENBAO_KEYS_FILE="${OPENBAO_KEYS_FILE:-}"
FORCE_ROTATE="${FORCE_ROTATE:-0}"

if [[ -z "${BAO_TOKEN:-}" ]]; then
  if [[ -z "$OPENBAO_KEYS_FILE" ]]; then
    OPENBAO_KEYS_FILE="$(find "$HOME/Descargas" "$HOME/.kube" -maxdepth 1 -type f -name '*vault*.json' 2>/dev/null | sort | tail -n1 || true)"
  fi
  if [[ -z "$OPENBAO_KEYS_FILE" || ! -f "$OPENBAO_KEYS_FILE" ]]; then
    echo "ERROR: Exporta BAO_TOKEN o OPENBAO_KEYS_FILE apuntando al JSON de inicialización de OpenBao." >&2
    exit 1
  fi
  BAO_TOKEN="$(jq -r '.root_token' "$OPENBAO_KEYS_FILE")"
fi

if [[ -z "$BAO_TOKEN" || "$BAO_TOKEN" == "null" ]]; then
  echo "ERROR: Token de OpenBao vacío o inválido." >&2
  exit 1
fi

# Pasa BAO_TOKEN y payload por stdin para no exponer secretos en argumentos de línea de comandos (ps/argv).
bao_exec_stdin() {
  {
    printf '%s\n' "$BAO_TOKEN"
    cat
  } | kubectl --kubeconfig "$KUBECONFIG_PATH" exec -i -n openbao openbao-0 -- sh -c '
    IFS= read -r BAO_TOKEN
    export BAO_TOKEN
    "$@"
  ' _ "$@"
}

bao_exec() {
  bao_exec_stdin "$@" < /dev/null
}

gen_hex() {
  openssl rand -hex "${1:-24}"
}

gen_b64() {
  openssl rand -base64 "${1:-32}" | tr -d '\n'
}

read_existing_key() {
  local path="$1"
  local field="$2"
  if [[ "$FORCE_ROTATE" == "1" ]]; then
    return 0
  fi
  bao_exec bao kv get -mount=secret -field="$field" "$path" 2>/dev/null || true
}

echo "1. Verificando estado de OpenBao..."
bao_exec bao status >/dev/null

echo "2. Verificando motor kv-v2 en secret/..."
if ! bao_exec bao secrets list -format=json | jq -e '."secret/"' >/dev/null 2>&1; then
  bao_exec bao secrets enable -path=secret kv-v2
fi

echo "3. Configurando autenticación Kubernetes..."
if ! bao_exec bao auth list -format=json | jq -e '."kubernetes/"' >/dev/null 2>&1; then
  bao_exec bao auth enable kubernetes
fi

bao_exec bao write auth/kubernetes/config \
  kubernetes_host="https://kubernetes.default.svc:443" \
  kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt >/dev/null

echo "4. Aplicando políticas HCL aisladas por entorno (QA, Prod, CI)..."
bao_exec_stdin bao policy write magi-qa-policy - >/dev/null <<'EOF'
path "secret/data/magi/qa/*" {
  capabilities = ["read"]
}
path "secret/metadata/magi/qa/*" {
  capabilities = ["list", "read"]
}
path "secret/data/magi/ci/harbor-pull" {
  capabilities = ["read"]
}
EOF

bao_exec_stdin bao policy write magi-prod-policy - >/dev/null <<'EOF'
path "secret/data/magi/prod/*" {
  capabilities = ["read"]
}
path "secret/metadata/magi/prod/*" {
  capabilities = ["list", "read"]
}
path "secret/data/magi/ci/harbor-pull" {
  capabilities = ["read"]
}
EOF

bao_exec_stdin bao policy write magi-ci-policy - >/dev/null <<'EOF'
path "secret/data/magi/ci/*" {
  capabilities = ["read"]
}
path "secret/metadata/magi/ci/*" {
  capabilities = ["list", "read"]
}
EOF

# Eliminar política compartida legada si existía
bao_exec bao policy delete magi-eso-policy >/dev/null 2>&1 || true

echo "5. Configurando roles de Kubernetes Auth dedicados para ESO..."
bao_exec bao delete auth/kubernetes/role/eso-openbao-role >/dev/null 2>&1 || true
bao_exec bao delete auth/kubernetes/role/magi-qa-role >/dev/null 2>&1 || true
bao_exec bao delete auth/kubernetes/role/magi-prod-role >/dev/null 2>&1 || true

bao_exec bao write auth/kubernetes/role/eso-qa-role \
  bound_service_account_names="external-secrets" \
  bound_service_account_namespaces="external-secrets" \
  policies="magi-qa-policy" \
  ttl="15m" >/dev/null

bao_exec bao write auth/kubernetes/role/eso-prod-role \
  bound_service_account_names="external-secrets" \
  bound_service_account_namespaces="external-secrets" \
  policies="magi-prod-policy" \
  ttl="15m" >/dev/null

bao_exec bao write auth/kubernetes/role/eso-ci-role \
  bound_service_account_names="external-secrets" \
  bound_service_account_namespaces="external-secrets" \
  policies="magi-ci-policy" \
  ttl="15m" >/dev/null

echo "6. Generando o conservando secretos criptográficos por entorno en OpenBao..."
QA_PG_PASS="${QA_PG_PASS:-$(read_existing_key magi/qa/app POSTGRES_PASSWORD)}"
QA_PG_PASS="${QA_PG_PASS:-$(gen_hex 24)}"
QA_NEXTAUTH_SECRET="${QA_NEXTAUTH_SECRET:-$(read_existing_key magi/qa/app NEXTAUTH_SECRET)}"
QA_NEXTAUTH_SECRET="${QA_NEXTAUTH_SECRET:-$(gen_b64 32)}"
QA_ZITADEL_CLIENT_ID="${QA_ZITADEL_CLIENT_ID:-$(read_existing_key magi/qa/app ZITADEL_CLIENT_ID)}"
QA_ZITADEL_CLIENT_ID="${QA_ZITADEL_CLIENT_ID:-magi-qa-web}"
QA_ZITADEL_CLIENT_SECRET="${QA_ZITADEL_CLIENT_SECRET:-$(read_existing_key magi/qa/app ZITADEL_CLIENT_SECRET)}"
QA_ZITADEL_CLIENT_SECRET="${QA_ZITADEL_CLIENT_SECRET:-$(gen_hex 24)}"

PROD_PG_PASS="${PROD_PG_PASS:-$(read_existing_key magi/prod/app POSTGRES_PASSWORD)}"
PROD_PG_PASS="${PROD_PG_PASS:-$(gen_hex 24)}"
PROD_NEXTAUTH_SECRET="${PROD_NEXTAUTH_SECRET:-$(read_existing_key magi/prod/app NEXTAUTH_SECRET)}"
PROD_NEXTAUTH_SECRET="${PROD_NEXTAUTH_SECRET:-$(gen_b64 32)}"
PROD_ZITADEL_CLIENT_ID="${PROD_ZITADEL_CLIENT_ID:-$(read_existing_key magi/prod/app ZITADEL_CLIENT_ID)}"
PROD_ZITADEL_CLIENT_ID="${PROD_ZITADEL_CLIENT_ID:-magi-prod-web}"
PROD_ZITADEL_CLIENT_SECRET="${PROD_ZITADEL_CLIENT_SECRET:-$(read_existing_key magi/prod/app ZITADEL_CLIENT_SECRET)}"
PROD_ZITADEL_CLIENT_SECRET="${PROD_ZITADEL_CLIENT_SECRET:-$(gen_hex 24)}"

jq -nc \
  --arg pg_pass "$QA_PG_PASS" \
  --arg db_url "postgres://magi:${QA_PG_PASS}@magi-postgres.magi-qa.svc.cluster.local:5432/magi_qa?sslmode=disable" \
  --arg nextauth_sec "$QA_NEXTAUTH_SECRET" \
  --arg z_cid "$QA_ZITADEL_CLIENT_ID" \
  --arg z_csec "$QA_ZITADEL_CLIENT_SECRET" \
  '{
    ENV: "qa",
    PORT: "8080",
    LOG_LEVEL: "debug",
    POSTGRES_USER: "magi",
    POSTGRES_PASSWORD: $pg_pass,
    POSTGRES_DB: "magi_qa",
    DATABASE_URL: $db_url,
    REDIS_URL: "redis://magi-redis.magi-qa.svc.cluster.local:6379",
    KAFKA_BROKERS: "magi-kafka.magi-qa.svc.cluster.local:9092",
    EXPORT_STORAGE_DIR: "/var/magi/exports",
    ZITADEL_DOMAIN: "zitadel.airesresearch.com",
    ZITADEL_ISSUER: "https://zitadel.airesresearch.com",
    ZITADEL_JWKS_URL: "https://zitadel.airesresearch.com/oauth/v2/keys",
    ZITADEL_AUDIENCE: "magi-backend",
    ZITADEL_CLIENT_ID: $z_cid,
    ZITADEL_CLIENT_SECRET: $z_csec,
    CORS_ORIGIN: "https://magi-qa.airesresearch.com",
    CORS_ALLOWED_ORIGINS: "https://magi-qa.airesresearch.com,http://magi-qa.local",
    NEXT_PUBLIC_API_URL: "https://api-qa.airesresearch.com",
    NEXTAUTH_URL: "https://magi-qa.airesresearch.com",
    NEXTAUTH_SECRET: $nextauth_sec
  }' | bao_exec_stdin bao kv put -mount=secret magi/qa/app - >/dev/null

jq -nc \
  --arg pg_pass "$PROD_PG_PASS" \
  --arg db_url "postgres://magi:${PROD_PG_PASS}@magi-postgres.magi-prod.svc.cluster.local:5432/magi_prod?sslmode=disable" \
  --arg nextauth_sec "$PROD_NEXTAUTH_SECRET" \
  --arg z_cid "$PROD_ZITADEL_CLIENT_ID" \
  --arg z_csec "$PROD_ZITADEL_CLIENT_SECRET" \
  '{
    ENV: "production",
    PORT: "8080",
    LOG_LEVEL: "info",
    LOG_FORMAT: "json",
    POSTGRES_USER: "magi",
    POSTGRES_PASSWORD: $pg_pass,
    POSTGRES_DB: "magi_prod",
    DATABASE_URL: $db_url,
    REDIS_URL: "redis://magi-redis.magi-prod.svc.cluster.local:6379",
    KAFKA_BROKERS: "magi-kafka.magi-prod.svc.cluster.local:9092",
    EXPORT_STORAGE_DIR: "/var/magi/exports",
    ZITADEL_DOMAIN: "zitadel.airesresearch.com",
    ZITADEL_ISSUER: "https://zitadel.airesresearch.com",
    ZITADEL_JWKS_URL: "https://zitadel.airesresearch.com/oauth/v2/keys",
    ZITADEL_AUDIENCE: "magi-backend",
    ZITADEL_CLIENT_ID: $z_cid,
    ZITADEL_CLIENT_SECRET: $z_csec,
    CORS_ORIGIN: "https://magi.airesresearch.com",
    CORS_ALLOWED_ORIGINS: "https://magi.airesresearch.com",
    NEXT_PUBLIC_API_URL: "https://api.airesresearch.com",
    NEXTAUTH_URL: "https://magi.airesresearch.com",
    NEXTAUTH_SECRET: $nextauth_sec
  }' | bao_exec_stdin bao kv put -mount=secret magi/prod/app - >/dev/null

echo "7. Configurando credencial Harbor Pull-Only (Robot Account) y CI..."
HARBOR_AUTH_B64="${HARBOR_AUTH_B64:-}"
if [[ -z "$HARBOR_AUTH_B64" && -f "$HOME/.docker/config.json" ]]; then
  HARBOR_AUTH_B64="$(jq -r '.auths["harbor.airesresearch.com"].auth // empty' "$HOME/.docker/config.json" 2>/dev/null || true)"
fi
if [[ -n "${HARBOR_USER:-}" && -n "${HARBOR_PASS:-}" && -z "$HARBOR_AUTH_B64" ]]; then
  HARBOR_AUTH_B64="$(printf '%s:%s' "$HARBOR_USER" "$HARBOR_PASS" | base64 -w0)"
fi
if [[ -z "$HARBOR_AUTH_B64" ]]; then
  echo "ERROR: Inicia sesión primero con 'docker login harbor.airesresearch.com' o exporta HARBOR_USER/HARBOR_PASS." >&2
  exit 1
fi

HARBOR_PULL_USER="$(read_existing_key magi/ci/harbor-pull HARBOR_PULL_USERNAME)"
HARBOR_PULL_PASS="$(read_existing_key magi/ci/harbor-pull HARBOR_PULL_PASSWORD)"
if [[ -z "$HARBOR_PULL_USER" || -z "$HARBOR_PULL_PASS" || "$FORCE_ROTATE" == "1" ]]; then
  EXISTING_ROBOT_ID="$(curl -s -H "Authorization: Basic $HARBOR_AUTH_B64" "https://harbor.airesresearch.com/api/v2.0/robots?q=name%3Dmagi-pull" | jq -r '.[0].id // empty')"
  if [[ -n "$EXISTING_ROBOT_ID" ]]; then
    curl -s -H "Authorization: Basic $HARBOR_AUTH_B64" -X DELETE "https://harbor.airesresearch.com/api/v2.0/robots/${EXISTING_ROBOT_ID}" >/dev/null
  fi
  ROBOT_RESP="$(curl -s -H "Authorization: Basic $HARBOR_AUTH_B64" -H "Content-Type: application/json" \
    -X POST "https://harbor.airesresearch.com/api/v2.0/robots" \
    -d '{"name":"magi-pull","level":"project","duration":-1,"disable":false,"permissions":[{"kind":"project","namespace":"magi","access":[{"resource":"repository","action":"pull"}]}]}')"
  HARBOR_PULL_USER="$(printf '%s' "$ROBOT_RESP" | jq -r '.name // empty')"
  HARBOR_PULL_PASS="$(printf '%s' "$ROBOT_RESP" | jq -r '.secret // empty')"
fi

if [[ -z "$HARBOR_PULL_USER" || -z "$HARBOR_PULL_PASS" ]]; then
  echo "ERROR: No se pudo crear/recuperar la cuenta Robot pull-only de Harbor." >&2
  exit 1
fi

PULL_AUTH_B64="$(printf '%s:%s' "$HARBOR_PULL_USER" "$HARBOR_PULL_PASS" | base64 -w0)"
DOCKER_PULL_CONFIG_JSON="$(jq -nc --arg auth "$PULL_AUTH_B64" '{
  auths: {
    "harbor.airesresearch.com": { auth: $auth }
  }
}')"

jq -nc \
  --arg u "$HARBOR_PULL_USER" \
  --arg p "$HARBOR_PULL_PASS" \
  --arg cfg "$DOCKER_PULL_CONFIG_JSON" \
  '{
    HARBOR_REGISTRY: "harbor.airesresearch.com",
    HARBOR_PULL_USERNAME: $u,
    HARBOR_PULL_PASSWORD: $p,
    DOCKER_CONFIG_JSON: $cfg
  }' | bao_exec_stdin bao kv put -mount=secret magi/ci/harbor-pull - >/dev/null

# Remover secreto legado magi/ci/harbor que contenía credenciales admin
bao_exec bao kv metadata delete -mount=secret magi/ci/harbor >/dev/null 2>&1 || true

echo "8. Configurando token del GitHub Runner..."
GITHUB_REPO_URL="${GITHUB_REPO_URL:-$(git config --get remote.origin.url | sed 's|\.git$||')}"
if [[ -z "${GITHUB_ACCESS_TOKEN:-}" ]]; then
  REPO_HOST_PATH="$(printf '%s' "$GITHUB_REPO_URL" | sed -E 's|^https?://github\.com/||')"
  GITHUB_ACCESS_TOKEN="$(printf 'protocol=https\nhost=github.com\npath=%s.git\n\n' "$REPO_HOST_PATH" | git -c credential.helper= -c credential.helper=store credential fill 2>/dev/null | sed -n 's/^password=//p' | head -n1 || true)"
fi
if [[ -z "${GITHUB_ACCESS_TOKEN:-}" ]]; then
  GITHUB_ACCESS_TOKEN="$(printf 'protocol=https\nhost=github.com\n\n' | git credential fill 2>/dev/null | sed -n 's/^password=//p' | head -n1 || true)"
fi
if [[ -z "${GITHUB_ACCESS_TOKEN:-}" ]]; then
  GITHUB_ACCESS_TOKEN="$(gh auth token 2>/dev/null || true)"
fi

if [[ -n "${GITHUB_ACCESS_TOKEN:-}" ]]; then
  jq -nc \
    --arg url "$GITHUB_REPO_URL" \
    --arg tok "$GITHUB_ACCESS_TOKEN" \
    '{ REPO_URL: $url, ACCESS_TOKEN: $tok }' | bao_exec_stdin bao kv put -mount=secret magi/ci/github-runner - >/dev/null
  echo "   -> Secreto secret/magi/ci/github-runner actualizado (sin exponer credenciales)."
else
  echo "AVISO: No se obtuvo token GitHub; exporta GITHUB_ACCESS_TOKEN con permisos de runner si deseas actualizarlo." >&2
fi

echo "✅ OpenBao endurecido y configurado exitosamente para MAGI (QA, Prod y CI)."
