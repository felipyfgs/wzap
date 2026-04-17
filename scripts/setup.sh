#!/bin/bash
# =============================================================================
# wzap — Build de imagens Docker (API Go + Web Nuxt)
# =============================================================================
# Targets disponíveis (no Dockerfile raiz):
#   combined   → API + Web numa única imagem (wzap:latest)            [default]
#   api-prod   → somente API Go (wzap-api:latest)
#   web-prod   → somente Web Nuxt (wzap-web:latest)
#
# Usage:
#   ./scripts/setup.sh                     # build combined (wzap:latest)
#   ./scripts/setup.sh --target=api-prod   # build só API
#   ./scripts/setup.sh --target=web-prod   # build só Web
#   ./scripts/setup.sh --split             # build api-prod + web-prod
#   ./scripts/setup.sh --push              # build + push ao Docker Hub
#   ./scripts/setup.sh --no-cache          # build sem cache
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

DOCKER_USER="${DOCKER_USERNAME:-felipyfgs17}"
GIT_SHA=$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo "local")

print_step() { echo -e "\n\033[1;34m▶ $*\033[0m"; }
print_ok()   { echo -e "\033[1;32m✔ $*\033[0m"; }
print_err()  { echo -e "\033[1;31m✖ $*\033[0m"; }
print_info() { echo -e "\033[0;37m  $*\033[0m"; }

AUTO_PUSH=false
NO_CACHE=""
TARGET="combined"
SPLIT=false

for arg in "$@"; do
    case "$arg" in
        --push)          AUTO_PUSH=true ;;
        --no-cache)      NO_CACHE="--no-cache" ;;
        --split)         SPLIT=true ;;
        --target=*)      TARGET="${arg#--target=}" ;;
    esac
done

if ! command -v docker &>/dev/null; then
    print_err "Docker não encontrado."
    exit 1
fi

# ─── Build helpers ───────────────────────────────────────────────────────────
# Args: <target> <local-tag> <hub-image>
build_target() {
    local target="$1"
    local local_tag="$2"
    local hub_image="$3"

    print_step "Build  target=${target}  [sha: ${GIT_SHA}]"
    print_info "Local  : ${local_tag}"
    print_info "Hub    : ${hub_image}:latest"

    # shellcheck disable=SC2086
    DOCKER_BUILDKIT=1 docker build $NO_CACHE \
        --target "${target}" \
        -t "${local_tag}" \
        -t "${hub_image}:latest" \
        -t "${hub_image}:${GIT_SHA}" \
        "$ROOT"
}

push_image() {
    local hub_image="$1"
    print_step "Push ${hub_image} → Docker Hub..."
    docker push "${hub_image}:latest"
    docker push "${hub_image}:${GIT_SHA}"
}

# ─── Build ────────────────────────────────────────────────────────────────────
if [[ "$SPLIT" == true ]]; then
    build_target "api-prod" "wzap-api:latest" "${DOCKER_USER}/wzap-api"
    build_target "web-prod" "wzap-web:latest" "${DOCKER_USER}/wzap-web"
else
    case "$TARGET" in
        combined) build_target "combined" "wzap:latest"     "${DOCKER_USER}/wzap" ;;
        api-prod) build_target "api-prod" "wzap-api:latest" "${DOCKER_USER}/wzap-api" ;;
        web-prod) build_target "web-prod" "wzap-web:latest" "${DOCKER_USER}/wzap-web" ;;
        *) print_err "Target desconhecido: ${TARGET}"; exit 1 ;;
    esac
fi

print_ok "Imagens prontas:"
docker images "${DOCKER_USER}/wzap*" \
    --format "  {{.Repository}}:{{.Tag}}\t{{.Size}}" | head -10

# ─── Push (opcional) ──────────────────────────────────────────────────────────
do_push() {
    if [[ "$SPLIT" == true ]]; then
        push_image "${DOCKER_USER}/wzap-api"
        push_image "${DOCKER_USER}/wzap-web"
    else
        case "$TARGET" in
            combined) push_image "${DOCKER_USER}/wzap" ;;
            api-prod) push_image "${DOCKER_USER}/wzap-api" ;;
            web-prod) push_image "${DOCKER_USER}/wzap-web" ;;
        esac
    fi
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
print_info "Para subir tudo (infra + api + web):"
print_info "  make docker-prod"
