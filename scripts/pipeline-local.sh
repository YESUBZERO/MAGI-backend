#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KUBECONFIG_PATH="${KUBECONFIG:-$HOME/.kube/config-k3s-tailscale}"
HARBOR_REGISTRY="${HARBOR_REGISTRY:-harbor.airesresearch.com}"
HARBOR_PROJECT="${HARBOR_PROJECT:-magi}"

cmd="${1:-ci}"
shift || true

validate_token() {
  local val="$1"
  if [[ ! "$val" =~ ^[a-zA-Z0-9._-]{1,64}$ ]]; then
    echo "ERROR: Valor inválido '$val'. Debe cumplir ^[a-zA-Z0-9._-]{1,64}$" >&2
    exit 1
  fi
}

run_ci() {
  echo "==> [1/3] Ejecutando Quality Gates Backend (Go vet + test -race)..."
  (cd "$ROOT_DIR/apps/server" && go vet ./... && go test -race ./...)

  echo "==> [2/3] Ejecutando Quality Gates Frontend (check-types + test)..."
  (cd "$ROOT_DIR/apps/web" && bun run check-types && bun run test)

  echo "==> [3/3] Validando manifiestos Kustomize (qa y prod)..."
  kubectl kustomize "$ROOT_DIR/deploy/k8s/overlays/qa" > /dev/null
  kubectl kustomize "$ROOT_DIR/deploy/k8s/overlays/prod" > /dev/null
  echo "✅ Quality Gates locales superados."
}

build_and_push() {
  local env_target="${1:-qa}"
  local tag="${2:-${env_target}-$(git -C "$ROOT_DIR" rev-parse --short HEAD)}"
  validate_token "$env_target"
  validate_token "$tag"

  local api_url="https://api-qa.airesresearch.com"
  if [[ "$env_target" == "prod" ]]; then
    api_url="https://api.airesresearch.com"
  fi

  local host_arch
  host_arch="$(uname -m)"
  if [[ "$host_arch" != "aarch64" && "$host_arch" != "arm64" ]] && ! docker buildx ls | grep -q "linux/arm64"; then
    echo "ℹ️ Host ${host_arch} sin emulación QEMU linux/arm64; despachando build nativo en Self-Hosted Runner ARM64 de k3s..."
    local repo_nwo gh_pat
    repo_nwo="$(git -C "$ROOT_DIR" config --get remote.origin.url | sed -E 's|^https?://github\.com/||; s|\.git$||')"
    gh_pat="${GH_TOKEN:-$(printf 'protocol=https\nhost=github.com\npath=%s.git\n\n' "$repo_nwo" | git credential fill 2>/dev/null | sed -n 's/^password=//p' | head -n1 || true)}"
    gh_pat="${gh_pat:-$(gh auth token 2>/dev/null || true)}"
    if [[ -n "$gh_pat" ]]; then
      GH_TOKEN="$gh_pat" gh workflow run ci-cd.yml -R "$repo_nwo" \
        --ref "$(git -C "$ROOT_DIR" rev-parse --abbrev-ref HEAD)" \
        -f "target_env=${env_target}" \
        -f "image_tag=${tag}"
      echo "✅ Workflow ARM64 despachado en GitHub Actions (env=${env_target}, tag=${tag})."
      return 0
    fi
    echo "ERROR: No se detectó soporte ARM64 local ni token GitHub para despachar el runner k3s." >&2
    exit 1
  fi

  echo "==> Compilando y subiendo imágenes linux/arm64 (env=${env_target}, tag=${tag})..."
  docker buildx build --platform linux/arm64 \
    -t "${HARBOR_REGISTRY}/${HARBOR_PROJECT}/server:${tag}" \
    -t "${HARBOR_REGISTRY}/${HARBOR_PROJECT}/server:${env_target}" \
    --push "$ROOT_DIR/apps/server"

  docker buildx build --platform linux/arm64 \
    -f "$ROOT_DIR/apps/web/Dockerfile" \
    --build-arg "NEXT_PUBLIC_API_URL=${api_url}" \
    -t "${HARBOR_REGISTRY}/${HARBOR_PROJECT}/web:${tag}" \
    -t "${HARBOR_REGISTRY}/${HARBOR_PROJECT}/web:${env_target}" \
    --push "$ROOT_DIR"

  echo "✅ Imágenes publicadas en ${HARBOR_REGISTRY}/${HARBOR_PROJECT} (tag=${tag})."
}

deploy_env() {
  local env_target="${1:-qa}"
  local ns="magi-${env_target}"
  local tag="${2:-${env_target}-$(git -C "$ROOT_DIR" rev-parse --short HEAD)}"
  validate_token "$env_target"
  validate_token "$tag"

  build_and_push "$env_target" "$tag"

  echo "==> Actualizando tag en deploy/k8s/overlays/${env_target}/kustomization.yaml..."
  sed -i -E "s|(newTag:[[:space:]]*).*|\\1${tag}|g" "$ROOT_DIR/deploy/k8s/overlays/${env_target}/kustomization.yaml"

  echo "==> Aplicando overlay directo y sincronizando ArgoCD (${ns})..."
  kubectl --kubeconfig "$KUBECONFIG_PATH" create namespace "$ns" --dry-run=client -o yaml | \
    kubectl --kubeconfig "$KUBECONFIG_PATH" apply -f -
  kubectl --kubeconfig "$KUBECONFIG_PATH" apply --server-side -k "$ROOT_DIR/deploy/k8s/overlays/${env_target}"
  kubectl --kubeconfig "$KUBECONFIG_PATH" -n "$ns" rollout restart deploy/magi-api deploy/magi-worker deploy/magi-web || true

  echo "✅ Despliegue en ${ns} disparado. Estado actual:"
  kubectl --kubeconfig "$KUBECONFIG_PATH" get pods,externalsecret,httproute -n "$ns"
}

cluster_status() {
  echo "=== ArgoCD Applications ==="
  kubectl --kubeconfig "$KUBECONFIG_PATH" get applications -n argocd
  echo ""
  echo "=== SecretStores & OpenBao ==="
  kubectl --kubeconfig "$KUBECONFIG_PATH" get clustersecretstore
  kubectl --kubeconfig "$KUBECONFIG_PATH" get pods -n openbao
  echo ""
  echo "=== GitHub Self-Hosted Runner ==="
  kubectl --kubeconfig "$KUBECONFIG_PATH" get pods -n github-runner
  echo ""
  echo "=== Entorno QA (magi-qa) ==="
  kubectl --kubeconfig "$KUBECONFIG_PATH" get pods,pvc,httproute,externalsecret -n magi-qa 2>/dev/null || echo "Namespace magi-qa aún no creado"
  echo ""
  echo "=== Entorno Prod (magi-prod) ==="
  kubectl --kubeconfig "$KUBECONFIG_PATH" get pods,pvc,httproute,externalsecret -n magi-prod 2>/dev/null || echo "Namespace magi-prod aún no creado"
}

case "$cmd" in
  ci)
    run_ci
    ;;
  build-push)
    build_and_push "${1:-qa}" "${2:-}"
    ;;
  deploy-qa)
    run_ci
    deploy_env "qa" "${1:-qa}"
    ;;
  deploy-prod)
    run_ci
    deploy_env "prod" "${1:-v0.2.0}"
    ;;
  status)
    cluster_status
    ;;
  *)
    echo "Uso: $0 {ci|build-push <qa|prod> [tag]|deploy-qa [tag]|deploy-prod [tag]|status}" >&2
    exit 1
    ;;
esac
