<!--
Sync Impact Report:
- Version change: 1.0.0 -> 1.1.0 (MINOR: principles materially expanded with concrete examples and new sections)
- Modified principles:
  - I. Arquitetura em Camadas (expanded with handler/service/repo patterns and file references)
  - II. Testes Obrigatorios (expanded with concrete test patterns)
  - III. Convencoes de Codigo Go (expanded with imports, naming, DTO patterns)
  - IV. Context e Erros (expanded with handler error boundary pattern)
  - V. Simplicidade (unchanged)
- Added sections:
  - Handlers (Padroes HTTP)
  - Services (Logica de Negocio)
  - Repos (Acesso a Dados)
  - DTOs (Request/Response)
  - Models (Dominio)
  - Frontend (Nuxt)
  - Banco de Dados (Migrations)
  - Seguranca
- Removed sections: none
- Templates requiring updates:
  - .specify/templates/spec-template.md: ✅ compatible (no changes needed)
  - .specify/templates/plan-template.md: ✅ compatible (no changes needed)
  - .specify/templates/tasks-template.md: ✅ compatible (no changes needed)
- Follow-up TODOs: none
-->

# wzap Constitution

## Principios Fundamentais

### I. Arquitetura em Camadas

Toda feature DEVE seguir handler -> service -> repo. Nunca pular camadas.

- **Handlers** (`internal/handler/`) recebem HTTP, parseiam DTOs, chamam services, retornam responses. Nenhuma logica de negocio.
- **Services** (`internal/service/`) contem logica de negocio, orquestram repos e engine whatsmeow. Nenhum SQL direto.
- **Repos** (`internal/repo/`) fazem queries SQL via pgx. Nenhuma logica de negocio.

Cada camada recebe dependencias via construtor:

```
NewSessionHandler(svc *service.SessionService, engine *wa.Manager) *SessionHandler
NewSessionService(repo *repo.SessionRepository, ...) *SessionService
NewSessionRepository(db *pgxpool.Pool) *SessionRepository
```

Referencias: `handler/session.go:16-26`, `service/session.go:22-34`, `repo/session.go:12-18`

### II. Testes Obrigatorios

Todo codigo novo DEVE ter testes.

- Usar `testing.T` padrao do Go. Nunca testify.
- Handler tests em pacote externo (`package handler_test`).
- Service tests podem usar pacote interno (`package service`) para acessar helpers unexported.
- Criar Fiber app por grupo de teste com `fiber.New(fiber.Config{DisableStartupMessage: true})`.
- Usar `httptest.NewRequest` + `app.Test(req, -1)`.
- Inicializar validator nos tests: `var _ = middleware.Validate`.
- Helpers em `internal/testutil/`: `NewApp()`, `DoRequest()`, `ParseResp()`.
- Rodar `go test -v -race ./...` antes de considerar pronto.

Referencias: `handler/session_test.go:1`, `handler/session_test.go:17-42`, `testutil/fiber.go`

### III. Convencoes de Codigo Go

- Imports em 3 grupos separados por linha em branco: stdlib, terceiros, interno (`wzap/internal/...`)
- Exportados: `PascalCase`. Unexported: `camelCase`
- Acronimos em maiuscula: `ID`, `URL`, `JID`, `NATS`, `S3`
- Construtores: `New<Type>(...)` retornando `*Type`
- JSON tags: `json:"camelCase"` com `omitempty` em opcionais
- Validate tags: `validate:"required"` em obrigatorios, `validate:"required,min=1"` em slices
- Erros wrapados: `fmt.Errorf("failed to X: %w", err)`
- Sem comentarios a nao ser que explicitamente pedido
- Logs via `internal/logger` (zerolog). Nunca `zerolog/log` direto.
- Slices inicializados com `make([]T, 0)` para evitar JSON null

Referencias: `dto/session.go:43-48`, `dto/message.go:31`

### IV. Context e Erros

- Passar `context.Context` como primeiro parametro em services e repos.
- Handlers usam `c.Context()` do Fiber.
- Handler eh o boundary de erro: retorna `dto.ErrorResp(title, msg)` com HTTP status apropriado.
- Services wrapam erros com contexto e retornam pra cima.
- Repos wrapam erros SQL com `fmt.Errorf("failed to <verb> <noun>: %w", err)`.
- Nunca expor detalhes internos de banco/service na resposta HTTP.

Referencias: `dto/response.go:32-45`, `handler/helpers.go:30-48`

### V. Simplicidade (YAGNI)

Comecar simples. Nao adicionar abstracao antes de precisar. Verificar codigo vizinho antes de introduzir nova dependencia. Nao criar interfaces quando um tipo concreto basta.

## Handlers (Padroes HTTP)

- Assinatura: `func (h *XxxHandler) MethodName(c *fiber.Ctx) error`
- Parse + validacao: `parseAndValidate(c, &req)` — combina BodyParser e struct validation
- Session ID: `mustGetSessionID(c)` apos `RequiredSession` middleware
- Sucesso: `c.JSON(dto.SuccessResp(data))` ou `c.Status(code).JSON(dto.SuccessResp(data))`
- Erro: `c.Status(code).JSON(dto.ErrorResp("Title", "message"))`
- Admin guard: `if c.Locals("authRole") != "admin" { return 403 }`
- Swagger godoc acima de cada handler: `@Summary`, `@Router`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Security`
- Update DTOs usam pointer fields: `*string`, `*SessionProxy` (nil = nao fornecido)

Referencias: `handler/session.go:28-53`, `handler/helpers.go:25-48`, `dto/session.go:80-84`

## Services (Logica de Negocio)

- Structs com campos unexported (repos, engine).
- UUIDs gerados com `uuid.NewString()`.
- Erros de validacao: `fmt.Errorf("name is required")` (sem wrap).
- Erros de infra: `fmt.Errorf("failed to create session: %w", err)` (com wrap).
- Logs nao-fatais: `logger.Warn().Err(err).Str("session", id).Msg("...")`.
- Model-to-DTO: funcoes helper como `SessionToResp(model, ...extra) -> DTO`.

Referencias: `service/session.go:36-63`, `service/session.go:198-200`

## Repos (Acesso a Dados)

- Structs com `*pgxpool.Pool` unexported.
- Queries SQL inline com backticks. Params posicionais: `$1, $2, ...`.
- Colunas nullable: `COALESCE(col, '')` no SELECT.
- JSONB colunas scaneadas diretamente nos structs (pgx serializa automaticamente).
- INSERT/UPDATE/DELETE usam `r.db.Exec(ctx, query, args...)`.
- SELECT single: `r.db.QueryRow(ctx, query, args...).Scan(...)`.
- SELECT many: `r.db.Query(ctx, query, args...)` com `defer rows.Close()`.
- Upsert: `ON CONFLICT ... DO NOTHING` ou `DO UPDATE`.
- Mensagens: paginacao com `limit/offset`, max 100.

Referencias: `repo/session.go:20-50`, `repo/message.go:24-26`, `repo/message.go:34-37`

## DTOs (Request/Response)

- Requests: `XxxReq` com tags `json` + `validate`.
- Responses: `XxxResp` com tags `json`.
- Response envelope: `dto.SuccessResp(data)` retorna `{success: true, data: ...}`.
- Error envelope: `dto.ErrorResp("Title", "msg")` retorna `{success: false, error: "Title", message: "msg"}`.
- Sub-DTOs para campos complexos: `SessionProxy`, `SessionSettings`, `ReplyContext`.
- Pointer fields para updates parciais: `*string`, `*SessionProxy`.

Referencias: `dto/response.go:3-13`, `dto/session.go:51-61`

## Models (Dominio)

- Structs planos com `json` tags espelhando o schema do banco.
- Value objects compartilhados entre model e dto: `SessionProxy`, `SessionSettings`.
- DTOs fazem cast: `SessionProxy(s.Proxy)`.
- Eventos: type alias `EventType = string` com constantes e mapa de validacao.
- Tabelas com prefixo `wz_`: `wz_sessions`, `wz_webhooks`, `wz_messages`.

Referencias: `model/session.go:22-34`, `model/events.go:3`, `model/events.go:155-199`

## Frontend (Nuxt)

- Tipos em `app/types/index.d.ts`
- Composables em `app/composables/` com `createSharedComposable` (VueUse) para singletons
- State SSR-safe: `useState<T>('key', () => default)`
- API client: `useWzap()` com `api<T>(path, options)` e auth automatico
- Componentes em `app/components/` com `<script setup lang="ts">`
- Validacao client-side com Zod (`z.object({...})`)
- Forms com `UForm` + `@submit` handler
- Notificacoes com `useToast()` — `toast.add({ title, color })`
- Emit de eventos: `defineEmits<{ created: [] }>()`
- Seguir padroes do Nuxt UI Dashboard (UDashboardPanel, UDashboardNavbar, UCard, etc.)
- Sem comentarios

## Banco de Dados (Migrations)

- Migrations em `internal/database/migrations/` com prefixo numerico: `001_schema.up.sql`
- Todas as tabelas com prefixo `wz_`
- Colunas em `snake_case`
- JSONB para campos complexos: `DEFAULT '{}'` ou `DEFAULT '[]'`
- Trigger `set_updated_at()` em todas as tabelas com `updated_at`
- Foreign keys com `ON DELETE CASCADE`
- Indexes unicos em `name`, `token`, `jid` (condicional para nullable)
- Indexes compostos para queries frequentes (mensagens por chat+timestamp)

## Seguranca

- Auth middleware le header `Authorization` (token puro, sem Bearer prefix)
- Comparacao constant-time para tokens
- Admin vs session role: `c.Locals("authRole")` e `c.Locals("sessionID")`
- Rate limit: 120 req/min por session ou IP
- Nunca logar ou commitar secrets — usar `.env` (git-ignored)

## Stack do Projeto

| Camada | Tecnologia |
|--------|-----------|
| Linguagem | Go 1.25 |
| HTTP | Fiber |
| WhatsApp Engine | whatsmeow |
| Banco de Dados | PostgreSQL (pgx) |
| Mensageria | NATS JetStream |
| Storage | MinIO (S3) |
| Logs | zerolog |
| Validacao | go-playground/validator |
| Frontend | Nuxt 3 + Nuxt UI Pro |

## Workflow de Desenvolvimento

1. Escrever spec do que quer construir
2. Criar plano tecnico respeitando as camadas (handler -> service -> repo)
3. Quebrar em tarefas pequenas
4. Implementar seguindo padroes existentes
5. Rodar `golangci-lint run ./...` e `go test -v -race ./...`
6. Rodar `go mod tidy` se adicionou dependencias
7. `make docs` se criou endpoints novos

## Governanca

Esta constituicao sobrepoe qualquer outra pratica. Mudancas exigem documentacao, justificativa e plano de migracao. Em duvida, seguir o padrao do codigo existente no projeto.

**Versao**: 1.1.0 | **Ratificada**: 2026-04-03 | **Ultima alteracao**: 2026-04-03
