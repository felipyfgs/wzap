#!/bin/bash
# =============================================================================
# wzap — Dev Setup: inicia backend (Go) + frontend (Nuxt) juntos
# =============================================================================
# Usage:
#   ./scripts/setup.sh          # modo dev (go run + nuxt dev)
#   ./scripts/setup.sh --build  # builda binário antes de iniciar
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${ROOT}/.env"

print_step() { echo -e "\n\033[1;34m▶ $*\033[0m"; }
print_ok()   { echo -e "\033[1;32m✔ $*\033[0m"; }
print_err()  { echo -e "\033[1;31m✖ $*\033[0m"; }
print_info() { echo -e "\033[0;37m  $*\033[0m"; }

BUILD_FIRST=false
for arg in "$@"; do
    case "$arg" in
        --build) BUILD_FIRST=true ;;
    esac
done

# ─── Pré-requisitos ───────────────────────────────────────────────────────────
for cmd in go node pnpm; do
    if ! command -v "$cmd" &>/dev/null; then
        print_err "Dependência ausente: $cmd"
        exit 1
    fi
done

# ─── Carrega .env ─────────────────────────────────────────────────────────────
if [[ -f "$ENV_FILE" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
    print_ok "Variáveis carregadas de ${ENV_FILE}"
else
    print_err ".env não encontrado em ${ROOT}"
    exit 1
fi

API_PORT="${PORT:-8080}"
WEB_PORT="${WEB_PORT:-3001}"

BACKEND_PID=""
FRONTEND_PID=""

# ─── Cleanup ao sair ──────────────────────────────────────────────────────────
cleanup() {
    echo ""
    print_step "Encerrando serviços..."
    [[ -n "$BACKEND_PID" ]]  && kill "$BACKEND_PID"  2>/dev/null || true
    [[ -n "$FRONTEND_PID" ]] && kill "$FRONTEND_PID" 2>/dev/null || true
    wait 2>/dev/null || true
    print_ok "Encerrado."
}
trap cleanup EXIT INT TERM

# ─── Backend ──────────────────────────────────────────────────────────────────
print_step "Iniciando backend Go (porta ${API_PORT})..."

cd "$ROOT"

if [[ "$BUILD_FIRST" == true ]]; then
    print_info "Compilando binário..."
    CGO_ENABLED=0 go build -o bin/wzap ./cmd/wzap/main.go
    ./bin/wzap &
else
    go run ./cmd/wzap/main.go &
fi

BACKEND_PID=$!
print_info "PID backend: ${BACKEND_PID}"

# Aguarda backend responder
print_info "Aguardando backend em http://localhost:${API_PORT}/health ..."
WAIT=0
until curl -sf "http://localhost:${API_PORT}/health" >/dev/null 2>&1; do
    sleep 1
    WAIT=$((WAIT + 1))
    if [[ $WAIT -ge 30 ]]; then
        print_err "Backend não respondeu em 30s. Verifique logs acima."
        exit 1
    fi
done
print_ok "Backend pronto."

# ─── Frontend ─────────────────────────────────────────────────────────────────
print_step "Iniciando frontend Nuxt (porta ${WEB_PORT})..."

cd "${ROOT}/web"

if [[ ! -d "node_modules" ]]; then
    print_info "Instalando dependências do frontend..."
    pnpm install --frozen-lockfile
fi

pnpm dev --dotenv "../.env" --port "${WEB_PORT}" &
FRONTEND_PID=$!
print_info "PID frontend: ${FRONTEND_PID}"

# ─── Resumo ───────────────────────────────────────────────────────────────────
echo ""
echo "============================================"
print_ok " wzap rodando"
print_info " API:      http://localhost:${API_PORT}"
print_info " Swagger:  http://localhost:${API_PORT}/swagger/"
print_info " Frontend: http://localhost:${WEB_PORT}"
echo "============================================"
echo " Pressione Ctrl+C para encerrar tudo."
echo ""

wait "$BACKEND_PID" "$FRONTEND_PID"
