#!/bin/bash
# =============================================================================
# wzap — Build & Push Docker Images
# =============================================================================
# Usage:
#   ./scripts/build.sh              # build api + web (prompt to push)
#   ./scripts/build.sh api          # build only api
#   ./scripts/build.sh web          # build only web
#   ./scripts/build.sh all --push   # build all + push without prompt
#   ./scripts/build.sh api --push   # build api + push without prompt
# =============================================================================

set -euo pipefail

DOCKER_USERNAME="felipyfgs17"
IMAGE_NAME="wzap"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

TARGET="${1:-all}"
AUTO_PUSH=false

for arg in "$@"; do
    case "$arg" in
        --push) AUTO_PUSH=true ;;
    esac
done

GIT_SHA=$(git -C "${PROJECT_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "local")

print_step() { echo -e "\n\033[1;34m▶ $*\033[0m"; }
print_ok()   { echo -e "\033[1;32m✔ $*\033[0m"; }
print_err()  { echo -e "\033[1;31m✖ $*\033[0m"; }
print_info() { echo -e "\033[0;37m  $*\033[0m"; }

build_api() {
    print_step "Building API image..."
    docker build --target prod \
        -t "${DOCKER_USERNAME}/${IMAGE_NAME}:api-latest" \
        -t "${DOCKER_USERNAME}/${IMAGE_NAME}:api-${GIT_SHA}" \
        "${PROJECT_ROOT}"
    print_ok "API image built → :api-latest / :api-${GIT_SHA}"
}

build_web() {
    print_step "Building Web image..."
    docker build --target web-prod \
        -t "${DOCKER_USERNAME}/${IMAGE_NAME}:web-latest" \
        -t "${DOCKER_USERNAME}/${IMAGE_NAME}:web-${GIT_SHA}" \
        -f "${PROJECT_ROOT}/web/Dockerfile" \
        "${PROJECT_ROOT}/web"
    print_ok "Web image built → :web-latest / :web-${GIT_SHA}"
}

push_images() {
    print_step "Pushing images to Docker Hub..."
    case "$TARGET" in
        api)
            docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:api-latest"
            docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:api-${GIT_SHA}"
            ;;
        web)
            docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:web-latest"
            docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:web-${GIT_SHA}"
            ;;
        all)
            docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:api-latest"
            docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:api-${GIT_SHA}"
            docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:web-latest"
            docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:web-${GIT_SHA}"
            ;;
    esac
    print_ok "Push complete!"
}

# ─── Main ─────────────────────────────────────────────────────────────────────
echo "============================================"
echo " wzap Docker Build  [sha: ${GIT_SHA}]"
echo "============================================"

case "$TARGET" in
    api) build_api ;;
    web) build_web ;;
    all) build_api; build_web ;;
    *)
        echo "Usage: $0 [api|web|all] [--push]"
        exit 1
        ;;
esac

echo ""
print_step "Images ready:"
docker images "${DOCKER_USERNAME}/${IMAGE_NAME}" \
    --format "  {{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | head -10

if [[ "$AUTO_PUSH" == true ]]; then
    push_images
else
    echo ""
    read -rp "Push to Docker Hub? (y/N): " confirm
    if [[ "${confirm,,}" == "y" ]]; then
        push_images
    else
        print_info "Skipping push."
    fi
fi
