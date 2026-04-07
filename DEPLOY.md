# wzap Docker Swarm Deployment Guide

## 🎯 Quick Start

### 1. Configurar Secrets no GitHub (para CI/CD automático)

Acesse: `Settings > Secrets and variables > Actions`

Adicione:
- `DOCKER_PASSWORD` = `@Contador10_` (sua senha Docker Hub)

### 2. Deploy Manual (primeira vez)

```bash
# Build e push local (opcional)
./build-and-push.sh

# Ou aguarde o CI/CD automático após push na main
```

### 3. Deploy no Swarm

```bash
# Inicializar Swarm (se ainda não iniciado)
docker swarm init

# Deploy da stack completa
./deploy-swarm.sh
```

## 📋 Estrutura de Imagens

| Serviço | Imagem | Tag |
|---------|--------|-----|
| API (Backend Go) | `felipyfgs17/wzap` | `api-latest`, `api-{sha}` |
| Web (Frontend Nuxt) | `felipyfgs17/wzap` | `web-latest`, `web-{sha}` |

## 🌐 URLs (com Traefik)

- **API**: http://api.wzap.local
- **Web Dashboard**: http://wzap.local
- **MinIO**: http://minio.wzap.local
- **MinIO Console**: http://minio-console.wzap.local

## ⚙️ Comandos Úteis

```bash
# Ver status dos serviços
docker service ls

# Ver logs de um serviço
docker service logs -f wzap_api

# Escalar serviço
docker service scale wzap_api=3

# Atualizar stack
docker stack deploy -c docker-compose.swarm.yml wzap

# Remover stack
docker stack rm wzap

# Visualizar redes
docker network ls

# Visualizar volumes
docker volume ls
```

## 🔧 Configuração de Secrets (Swarm)

```bash
# Criar secrets manualmente
echo "senha_postgres" | docker secret create wzap_postgres_password -
echo "admin123" | docker secret create wzap_minio_access_key -
echo "secret123" | docker secret create wzap_minio_secret_key -
echo "token_admin" | docker secret create wzap_admin_token -
```

## 📊 Recursos Alocados

| Serviço | Replicas | CPU Limit | Mem Limit |
|---------|----------|-----------|-----------|
| API | 2 | 1.0 | 512M |
| Web | 2 | 0.5 | 256M |
| Postgres | 1 | 1.0 | 1G |
| MinIO | 1 | 0.5 | 512M |
| NATS | 1 | 0.5 | 256M |
| Redis | 1 | 0.25 | 128M |

## 🔄 CI/CD Pipeline

O GitHub Actions faz:
1. Testes e lint
2. Build das imagens
3. Push para Docker Hub
4. Tags automáticas por commit SHA

Trigger: Push na branch `main` ou tags `v*`
