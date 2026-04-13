## Context

O wzap usa zerolog como biblioteca de logging. A configuração atual em `internal/logger/logger.go` inicializa um `zerolog.ConsoleWriter` (dev) ou JSON writer (prod) com `.With().Timestamp().Logger()`. O formato de timestamp padrão do ConsoleWriter é `time.Kitchen` (`3:04PM`), sem data.

Os logs são emitidos diretamente via `logger.Info()`, `logger.Debug()`, etc. em todos os módulos. Não existe padronização de campos estruturados — cada módulo usa convenções próprias. O Chatwoot usa prefixo `[CW]` na mensagem, enquanto os demais módulos não possuem identificação de origem.

Atualmente ~80+ pontos de log no Chatwoot, ~60+ no wa/events, ~30+ no webhook dispatcher, ~10+ em middleware/handler, e ~10+ em infra (db, nats, minio).

## Goals / Non-Goals

**Goals:**
- Timestamp completo em todos os ambientes (data + hora)
- Campo `component` estruturado para identificar origem de cada log
- Campos consistentes entre módulos (ex: sempre `session`, nunca `sessionID`)
- Enriquecer logs de eventos WhatsApp com dados úteis (msgType, mediaType)
- Unificar logs redundantes no webhook dispatcher
- Adicionar contexto faltante em logs de panic recovery e dispatch

**Non-Goals:**
- Migrar de zerolog para outra biblioteca
- Implementar tracing distribuído (OpenTelemetry)
- Adicionar request ID / correlation ID
- Alterar nível de log padrão da aplicação
- Alterar formato JSON de produção (apenas enriquecer campos)

## Decisions

### 1. Campo `component` via sub-logger vs campo inline

**Decisão**: Usar `.Str("component", "xxx")` inline em cada chamada de log.

**Alternativa considerada**: Criar sub-loggers por módulo (`logger.With().Str("component", "wa").Logger()`) e injetar via construtor.

**Razão**: Sub-loggers exigiriam mudança na assinatura de todos os construtores e armazenamento em structs. O campo inline é menos invasivo, mantém a API atual do `internal/logger`, e pode ser migrado para sub-loggers no futuro sem breaking changes. A consistência será garantida via convenção e code review.

**Valores de component**:
| Módulo | Valor |
|---|---|
| `internal/wa/` | `wa` |
| `internal/integrations/chatwoot/` | `chatwoot` |
| `internal/webhook/` | `webhook` |
| `internal/middleware/logger.go` | `http` |
| `internal/middleware/recovery.go` | `http` |
| `internal/websocket/` | `ws` |
| `internal/broker/` | `nats` |
| `internal/storage/` | `s3` |
| `internal/database/` | `db` |
| `internal/server/` | `server` |
| `internal/handler/` | `handler` |
| `internal/service/` | `service` |

### 2. Remoção de prefixos `[CW]` da Msg

**Decisão**: Remover `[CW]` do campo Msg e usar `.Str("component", "chatwoot")` em vez disso.

**Razão**: Tags na mensagem não são filtráveis por ferramentas estruturadas (jq, Loki, Grafana). O campo `component` permite queries como `{component="chatwoot"}` diretamente.

### 3. Formato de timestamp

**Decisão**:
- Dev (ConsoleWriter): `2006-01-02 15:04:05` — legível no terminal
- Prod (JSON): `time.RFC3339` — padrão ISO 8601, compatível com todas as ferramentas de ingestão

### 4. Enriquecimento de Message received

**Decisão**: Adicionar `msgType` e `mediaType` (quando aplicável) ao log `"Message received"` em `wa/events.go`, aproveitando a função `extractMessageContent` que já existe.

### 5. Unificação de logs de webhook dispatch

**Decisão**: Juntar os dois logs separados ("Dispatching webhook" + "Active webhooks found") em um único log que inclui `session`, `event`, `webhooks` (count) e `globalURL`.

## Risks / Trade-offs

- **[Volume de mudanças]** → Cada módulo será alterado em commits separados por área, permitindo revert granular.
- **[Quebra de filtros existentes]** → Documentar a migração de `grep "[CW]"` para `jq '.component=="chatwoot"'` ou `{component="chatwoot"}` no Loki.
- **[Performance]** → Adicionar 1 campo `.Str("component", "xxx")` por log tem overhead negligível (zerolog é zero-allocation para campos ignorados pelo nível).
- **[Impacto em métricas Prometheus]** → Nenhum. As mudanças são apenas no sistema de logging, métricas permanecem inalteradas.
- **[Impacto em webhooks]** → Nenhum. Os logs não afetam o payload dos webhooks.

## Open Questions

- Nenhuma questão aberta. As decisões técnicas são de baixo risco e facilmente reversíveis.
