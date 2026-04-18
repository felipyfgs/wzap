# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Um documento complementar mais detalhado está em [AGENTS.md](AGENTS.md) — consulte-o para convenções de nomeação de arquivos, padrões de handler/service, idiomas de DTO/repo e contexto completo de Docker/Chatwoot. Este arquivo cobre apenas o que é mais estrutural.

## Idioma (obrigatório)

**Sempre** converse em **Português do Brasil** — com o usuário e entre agentes/subagents. Isso se aplica a:

- Todas as respostas ao usuário.
- Prompts passados para subagents (Agent tool) e respostas recebidas deles.
- Mensagens de commit, descrições de PR, comentários em issues.
- Texto em planos, TODOs e updates intermediários.

Exceções: identificadores de código, nomes de comandos, logs técnicos e trechos citados de ferramentas externas permanecem no idioma original. Comentários já existentes no código podem estar em PT ou EN — **preserve o idioma original** ao editar.

## Commands

```bash
# Go backend
make dev                # go run cmd/wzap/main.go
make build              # CGO_ENABLED=0 → bin/wzap
make docs               # regenerate Swagger (swag init)
make tidy
make install-tools      # golangci-lint v2.11.4 + swag

# Tests (sem .golangci.yml no repo — defaults)
go test -v -race ./...
go test -v -race ./internal/service/...              # pacote específico
go test -v -race -run TestFunctionName ./...          # teste específico
golangci-lint run ./...

# Frontend (Nuxt 4 SPA, ssr: false)
make web-install        # pnpm install (cd web)
make web-dev            # pnpm dev

# Docker (infra + serviços em camadas)
make docker-dev         # compose.yml + compose.dev.yml (air + nuxt hot reload)
make docker-prod        # compose.yml + compose.prod.yml (compilado)
make chatwoot-up        # docker/chatwoot/docker-compose.yml (usa rede externa wzap_chatwoot)
```

Testes de DB precisam de `DATABASE_URL` apontando para um Postgres acessível. Regenerar Swagger é necessário após adicionar/alterar anotações de um handler exportado — não há hook de pre-commit para isso.

## Architecture — partes que exigem leitura de múltiplos arquivos

### Layering e wiring

`cmd/wzap/main.go` → carrega config → abre pool pgx + roda migrations embutidas (`migrations/*.sql` via `//go:embed`) → conecta NATS JetStream + MinIO → `server.New(cfg, db, nats, minio)` → `SetupRoutes()`.

`SetupRoutes` em [internal/server/router.go](internal/server/router.go) é o root da DI. Constrói repos → services → handlers manualmente (sem framework). A **ordem de registro de rotas importa**: rotas da Cloud API (`/v\d+\.\d+/...`) são registradas ANTES do grupo de auth, pois precisam ficar acessíveis sem o middleware de token admin/session.

Camadas são estritas: `handler → service → repo`. Handlers fazem parse HTTP + retornam DTOs; services concentram lógica de negócio; repos usam SQL bruto com params posicionais (`$1`, `$2`) e listas de colunas em constantes de pacote. Não cruze camadas.

### Dois engines de WhatsApp

O sistema fala WhatsApp por dois caminhos, e cada sessão pode usar qualquer um dos dois:

1. **whatsmeow** (protocolo multi-device direto) — gerenciado por [internal/wa/](internal/wa/). `wa.Manager` mantém um mapa de clients protegido por `sync.RWMutex` e é dono do ciclo de vida da sessão (QR, connect, disconnect, handlers de evento).
2. **Emulador da Cloud API** — [internal/handler/cloud_api.go](internal/handler/cloud_api.go) emula a WhatsApp Cloud API do Facebook para que o modo Cloud inbox do Chatwoot converse com o wzap como se fosse a Meta. Essas rotas **nunca** retornam 401 (dispararia `reauthorization_required` no Chatwoot); `warnTokenMismatch` apenas loga e segue.

Services que precisam funcionar nos dois engines usam um dispatcher genérico: `runSessionRuntime[T any](...)` + runtimes cacheados em contexto via `context.WithValue(ctx, runtimeCtxKey{}, r)`. Ao adicionar um novo tipo de mensagem ou capability, verifique se os dois engines precisam suportar — capability faltante vira `CapabilityError` (com `Unwrap()`).

### Integração Chatwoot

[internal/integrations/chatwoot/](internal/integrations/chatwoot/) é o subsistema arquiteturalmente mais denso. Os prefixos de arquivo que sobraram após o refactor são estruturais:

- `inbox_*` = abstração do modo de inbox (`inbox.go`, `inbox_api.go`, `inbox_cloud.go`, `inbox_common.go`)
- `wa_events*` = pipeline de eventos vindos do WhatsApp
- Todos os outros arquivos (webhook_outbound, conversation, bot, labels, backfill, mapping, ...) são puramente Chatwoot-side — o antigo prefixo `cw_*` foi removido em `2edce63`; não reintroduza.

A interface `InboxHandler` em [inbox.go](internal/integrations/chatwoot/inbox.go) tem duas implementações (`apiInboxHandler`, `cloudInboxHandler`), escolhidas por sessão via `cfg.InboxType`. `Service.processMessage` roteia para a implementação certa.

Para `inbox_type=cloud`, `wz_chatwoot.database_uri` (Postgres do próprio Chatwoot) permite lookup direto de `messages.source_id` para mapeamento de WAID — evita corrida com timing de webhook. `POST /sessions/{sessionId}/integrations/chatwoot/backfill` preenche retroativamente `wz_messages.cw_message_id/cw_conversation_id/cw_source_id`. Nunca logue o `database_uri` completo.

### Fluxo de eventos e trabalho assíncrono

Fan-out de webhook/WebSocket passa por [internal/webhook/](internal/webhook/) (dispatcher HTTP) + [internal/websocket/](internal/websocket/) (hub) + [internal/broker/](internal/broker/) (NATS JetStream). Trabalho em background (entrega de webhook, upload de mídia, sync de histórico) roda em workers `async.Pool` que drenam graciosamente no shutdown — não solte goroutines cruas para nada que possa sobreviver além do request.

Enums de domínio tipados (`EventType`, `EventCategory`, `EngineCapability`) vivem em [internal/model/](internal/model/) — prefira-os antes de inventar novas constantes string.

### Auth

Token admin (`ADMIN_TOKEN` env) é comparado com `crypto/subtle.ConstantTimeCompare`. API keys por-sessão ficam guardadas na sessão e são resolvidas pelo `middleware.Auth` após falha da checagem admin. Escopos: `admin` (tudo) vs `session` (uma sessão, aplicado por `RequiredSession`). Modo de auth do WebSocket é configurável via `WS_AUTH_MODE` (header ou query param).

### Frontend ↔ backend

SPA Nuxt 4 em `web/`. Código server-side (`web/server/api/[...].ts`, `web/server/routes/ws.ts`) faz proxy para a API Go usando `NUXT_API_URL` (server-only — nunca exposto ao browser). Acesso do browser ao MinIO usa um endpoint separado com whitelist (`NUXT_MINIO_ENDPOINT`) para evitar SSRF.

## Convenções importantes antes de editar

- **Naming de handler vs service**: `Handle*` exportado (handlers Fiber) vs `process*` não-exportado (internals de service). Não misture.
- **Padrão de handler**: `getSessionID(c)` → `parseAndValidate(c, &req)` → chamada de service → `dto.SuccessResp(...)` / `dto.ErrorResp(...)`. 500 retorna `"internal server error"` genérico; 400 pode incluir `err.Error()`.
- **Padrão de repo**: funções `scanXxx(scanner, &m)` dedicadas, apoiadas por uma interface local `xxxScanner`, para que `pgx.Row` e `pgx.Rows` reusem a mesma lógica. Sentinel errors (`ErrSessionNotFound`, etc.) são double-wrapped: `fmt.Errorf("%w: %w", ErrNotFound, err)`.
- **Logging**: apenas o singleton `logger` — toda linha começa com `.Str("component", "xxx")`. Sem `log.Print` / `fmt.Print`.
- **Models vs DTOs**: modelos em `internal/model/` não têm tags de validação; DTOs de request em `internal/dto/` têm (`validate:"required"` etc.). Mappers ficam no pacote `dto` (`SessionToResp(...)`).
- **DTOs de update usam ponteiros** (`*string`, `*bool`) para updates parciais — aplique com nil-check, não com checagem de zero-value.

## OpenSpec

O diretório [openspec/](openspec/) guarda propostas de mudança (`changes/`) e specs vigentes (`specs/`) num workflow spec-driven. A ordem exigida por `openspec/config.yaml` para implementar uma change é `model → repo → service → handler → tests`, finalizando com `golangci-lint run ./...` e `go test -race ./...`. Ao trabalhar numa change ativa, prefira os skills `openspec-*` em vez de editar os arquivos na mão.
