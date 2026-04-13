## Why

Os logs do wzap atualmente apresentam problemas de observabilidade: timestamps sem data (apenas hora no formato `3:04PM`), ausência de campo estruturado `component` para identificar a origem do log, campos inconsistentes entre módulos (`sessionID` vs `session`), logs órfãos sem contexto de sessão, e eventos WhatsApp que não logam dados úteis do evento recebido (apenas o ID). Isso dificulta debugging em produção, filtragem via Grafana/Loki, e correlação de eventos entre componentes.

## What Changes

- **Logger base**: Adicionar timestamp completo (RFC3339 em produção, `2006-01-02 15:04:05` em dev)
- **Campo `component` estruturado**: Substituir prefixos hardcoded `[CW]` na Msg por `.Str("component", "cw")` em todos os logs do Chatwoot, e adicionar `component` nos demais módulos (`wa`, `webhook`, `http`, `ws`, `db`, `nats`, `s3`)
- **Padronização de campos**: Corrigir `sessionID` → `session` em `handler/message.go`; adicionar `session` em logs órfãos do webhook dispatcher
- **Enriquecimento de eventos WA**: Adicionar `msgType` e `mediaType` no log de Message received; adicionar dados úteis em eventos DEBUG (callId, newsletter JID, mediaRetry msgId, etc.)
- **Unificação de logs de dispatch**: Juntar "Dispatching webhook" + "Active webhooks found" em um único log
- **Recovery middleware**: Adicionar `method`, `path`, `ip` no log de panic recovered
- **HTTP middleware**: Adicionar `component=http`
- **Correção de níveis**: `AppStateSyncError` de DEBUG → WARN

## Não-objetivos

- Não será alterado o backend de logging (zerolog permanece)
- Não será implementado tracing distribuído (OpenTelemetry) nesta change
- Não será adicionado request ID / correlation ID (escopo separado)
- Não será alterada a estrutura de logs JSON em produção (apenas enriquecimento de campos)
- Não será alterado o nível de log padrão da aplicação

## Capabilities

### New Capabilities
- `structured-log-format`: Padronização do formato de logs com timestamp completo, campo `component` estruturado, e campos consistentes em todos os módulos
- `enriched-event-logging`: Enriquecimento dos logs de eventos WhatsApp com dados úteis (msgType, mediaType, callId, newsletter JID, etc.)

### Modified Capabilities

## Impact

- **Arquivos afetados**: `internal/logger/logger.go`, `internal/wa/events.go`, `internal/webhook/dispatcher.go`, `internal/middleware/logger.go`, `internal/middleware/recovery.go`, `internal/handler/message.go`, `internal/integrations/chatwoot/` (~15 arquivos com `[CW]`), `internal/websocket/hub.go`, `internal/broker/nats.go`, `internal/storage/minio.go`, `internal/database/postgres.go`, `internal/server/server.go`, `internal/server/router.go`
- **APIs**: Nenhuma alteração de API. Mudanças são internas ao sistema de logging.
- **Dependências**: Nenhuma nova dependência. Apenas zerolog existente.
- **Breaking changes**: Nenhum. Logs JSON em produção terão campos adicionais (retrocompatível).

## Riscos e Mitigações

- **Risco**: Volume de mudanças espalhado por muitos arquivos pode introduzir regressões.
  - **Mitigação**: Cada módulo será alterado de forma independente e testado isoladamente.
- **Risco**: Remoção de `[CW]` pode quebrar alertas/filtros existentes baseados em grep.
  - **Mitigação**: O campo `component=cw` fornece filtragem equivalente e mais robusta.
