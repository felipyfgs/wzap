#!/bin/bash
# =============================================================================
# wzap — Build da imagem combinada (API Go + Web Nuxt)
# =============================================================================
# Infraestrutura (postgres, minio, nats) é gerenciada separadamente:
#   docker compose up -d        → sobe infra + app
#   docker compose up -d --build → rebuild + sobe tudo
#
# Usage:
#   ./scripts/setup.sh            # build da imagem wzap:latest
#   ./scripts/setup.sh --push     # build + push para Docker Hub
#   ./scripts/setup.sh --no-cache # build sem cache
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

DOCKER_USER="${DOCKER_USERNAME:-felipyfgs17}"
IMAGE="${DOCKER_USER}/wzap"
GIT_SHA=$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo "local")

print_step() { echo -e "\n\033[1;34m▶ $*\033[0m"; }
print_ok()   { echo -e "\033[1;32m✔ $*\033[0m"; }
print_err()  { echo -e "\033[1;31m✖ $*\033[0m"; }
print_info() { echo -e "\033[0;37m  $*\033[0m"; }

AUTO_PUSH=false
NO_CACHE=""

for arg in "$@"; do
    case "$arg" in
        --push)     AUTO_PUSH=true ;;
        --no-cache) NO_CACHE="--no-cache" ;;
    esac
done

if ! command -v docker &>/dev/null; then
    print_err "Docker não encontrado."
    exit 1
fi

# ─── Build ────────────────────────────────────────────────────────────────────
print_step "Build da imagem combinada  [sha: ${GIT_SHA}]"
print_info "Target : combined"
print_info "Context: ${ROOT}"

# shellcheck disable=SC2086
docker build $NO_CACHE \
    --target combined \
    -t "${IMAGE}:latest" \
    -t "${IMAGE}:${GIT_SHA}" \
    -t "wzap:latest" \
    "$ROOT"

print_ok "Imagem pronta:"
docker images "${IMAGE}" \
    --format "  {{.Repository}}:{{.Tag}}\t{{.Size}}" | head -5

# ─── Push (opcional) ──────────────────────────────────────────────────────────
do_push() {
    print_step "Push para Docker Hub..."
    docker push "${IMAGE}:latest"
    docker push "${IMAGE}:${GIT_SHA}"
    print_ok "Push concluído."
}

if [[ "$AUTO_PUSH" == true ]]; then
    do_push
else
    echo ""
    read -rp "Push para Docker Hub? (y/N): " confirm
    [[ "${confirm,,}" == "y" ]] && do_push || print_info "Push ignorado."
fi

echo ""
print_info "Para subir tudo:"
print_info "  docker compose up -d"
