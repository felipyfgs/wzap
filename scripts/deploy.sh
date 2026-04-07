#!/bin/bash
# =============================================================================
# wzap — Deploy to Docker Swarm
# =============================================================================
# Usage:
#   ./scripts/deploy.sh           # deploy stack completo
#   ./scripts/deploy.sh --pull    # pull imagens antes de deployar
#   ./scripts/deploy.sh --status  # exibir status atual sem deployar
# =============================================================================

set -euo pipefail

DOCKER_USERNAME="felipyfgs17"
IMAGE_NAME="wzap"
STACK_NAME="wzap"
STACK_FILE="stacks/wzap-complete.yml"

print_step() { echo -e "\n\033[1;34m▶ $*\033[0m"; }
print_ok()   { echo -e "\033[1;32m✔ $*\033[0m"; }
print_info() { echo -e "\033[0;37m  $*\033[0m"; }
print_warn() { echo -e "\033[1;33m⚠ $*\033[0m"; }

show_status() {
    print_step "Stack: ${STACK_NAME}"
    docker stack ps "${STACK_NAME}" \
        --format "  {{.Name}}\t{{.CurrentState}}\t{{.Error}}" 2>/dev/null || \
        print_info "Stack não encontrado."

    echo ""
    print_step "Services:"
    docker service ls --filter "label=com.docker.stack.namespace=${STACK_NAME}" \
        --format "  {{.Name}}\t{{.Replicas}}\t{{.Image}}" 2>/dev/null || true
}

pull_images() {
    print_step "Pulling latest images..."
    docker pull "${DOCKER_USERNAME}/${IMAGE_NAME}:api-latest"
    docker pull "${DOCKER_USERNAME}/${IMAGE_NAME}:web-latest"
    print_ok "Images updated."
}

deploy() {
    if ! docker info --format '{{.Swarm.LocalNodeState}}' | grep -q "active"; then
        print_warn "Docker Swarm não está ativo. Inicializando..."
        docker swarm init --advertise-addr "$(hostname -i | awk '{print $1}')"
    fi

    if ! docker network ls --format '{{.Name}}' | grep -q "^traefik_public$"; then
        print_step "Criando rede traefik_public..."
        docker network create --driver=overlay --attachable traefik_public
        print_ok "Rede criada."
    fi

    if [[ ! -f "${STACK_FILE}" ]]; then
        echo "Erro: stack file '${STACK_FILE}' não encontrado."
        exit 1
    fi

    print_step "Deploying stack '${STACK_NAME}' via ${STACK_FILE}..."
    docker stack deploy -c "${STACK_FILE}" "${STACK_NAME}" --with-registry-auth
    print_ok "Deploy enviado!"

    echo ""
    print_step "Aguardando serviços (15s)..."
    sleep 15
    show_status

    echo ""
    print_ok "Deploy concluído!"
    echo ""
    print_info "URLs:"
    print_info "  API    → https://api.wzap.gacont.com.br"
    print_info "  Web    → https://app.wzap.gacont.com.br"
    print_info "  MinIO  → https://s3.wzap.gacont.com.br"
    echo ""
    print_info "Comandos úteis:"
    print_info "  Logs API:   docker service logs -f ${STACK_NAME}_wzap_api"
    print_info "  Logs Web:   docker service logs -f ${STACK_NAME}_wzap_web"
    print_info "  Remover:    docker stack rm ${STACK_NAME}"
}

# ─── Entry point ──────────────────────────────────────────────────────────────
echo "============================================"
echo " wzap Swarm Deploy"
echo "============================================"

case "${1:-deploy}" in
    --status)  show_status ;;
    --pull)    pull_images; deploy ;;
    deploy|"") deploy ;;
    *)
        echo "Usage: $0 [--pull | --status]"
        exit 1
        ;;
esac
