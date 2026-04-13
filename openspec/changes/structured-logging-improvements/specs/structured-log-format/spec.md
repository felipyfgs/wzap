## ADDED Requirements

### Requirement: Timestamp completo em todos os logs
O sistema DEVE incluir data e hora completa em todos os logs. Em ambiente de desenvolvimento (ConsoleWriter), o formato DEVE ser `2006-01-02 15:04:05`. Em produção (JSON), o campo `time` DEVE usar formato RFC3339 (`2006-01-02T15:04:05Z07:00`).

#### Scenario: Log em ambiente de desenvolvimento
- **WHEN** o logger é inicializado com `environment=development`
- **THEN** o timestamp exibido no terminal DEVE seguir o formato `2006-01-02 15:04:05` (ex: `2026-04-13 19:51:00`)

#### Scenario: Log em ambiente de produção
- **WHEN** o logger é inicializado com `environment=production`
- **THEN** o campo `time` no JSON DEVE seguir RFC3339 (ex: `2026-04-13T19:51:00-03:00`)

### Requirement: Campo component estruturado em todos os logs
Todo log emitido DEVE incluir o campo estruturado `component` com o identificador do módulo de origem. Os valores DEVEM ser: `wa`, `chatwoot`, `webhook`, `http`, `ws`, `nats`, `s3`, `db`, `server`, `handler`, `service`.

#### Scenario: Log do módulo Chatwoot
- **WHEN** um log é emitido por qualquer arquivo em `internal/integrations/chatwoot/`
- **THEN** o log DEVE conter `.Str("component", "chatwoot")` e NÃO DEVE conter o prefixo `[CW]` na mensagem

#### Scenario: Log do módulo WhatsApp engine
- **WHEN** um log é emitido por qualquer arquivo em `internal/wa/`
- **THEN** o log DEVE conter `.Str("component", "wa")`

#### Scenario: Log do middleware HTTP
- **WHEN** um log é emitido por `internal/middleware/logger.go` ou `internal/middleware/recovery.go`
- **THEN** o log DEVE conter `.Str("component", "http")`

#### Scenario: Log do webhook dispatcher
- **WHEN** um log é emitido por qualquer arquivo em `internal/webhook/`
- **THEN** o log DEVE conter `.Str("component", "webhook")`

#### Scenario: Log do WebSocket hub
- **WHEN** um log é emitido por qualquer arquivo em `internal/websocket/`
- **THEN** o log DEVE conter `.Str("component", "ws")`

#### Scenario: Log de infraestrutura (db, nats, s3)
- **WHEN** um log é emitido por `internal/database/`, `internal/broker/`, ou `internal/storage/`
- **THEN** o log DEVE conter `.Str("component", "db")`, `"nats"`, ou `"s3"` respectivamente

### Requirement: Padronização do campo de sessão
Todos os logs que referenciam uma sessão DEVEM usar o campo `session` (não `sessionID`, não `sessionId`).

#### Scenario: Log em handler/message.go
- **WHEN** o handler de mensagem emite um log com ID da sessão
- **THEN** o campo DEVE ser `.Str("session", sessionID)` e NÃO `.Str("sessionID", sessionID)`

#### Scenario: Todos os módulos usam campo consistente
- **WHEN** qualquer módulo emite um log referenciando uma sessão
- **THEN** o campo DEVE ser `session` em todos os casos

### Requirement: Logs de webhook dispatch unificados
O webhook dispatcher DEVE emitir um único log ao iniciar o dispatch de um evento, contendo todos os campos relevantes. Os dois logs separados ("Dispatching webhook" e "Active webhooks found") DEVEM ser substituídos por um único log.

#### Scenario: Dispatch de webhook para sessão
- **WHEN** o dispatcher processa um evento para uma sessão
- **THEN** DEVE emitir um único log DEBUG com campos `session`, `event`, `webhooks` (count de webhooks ativos), e `globalURL` (URL do webhook global, se configurado)

### Requirement: Contexto completo no log de panic recovery
O middleware de recovery DEVE incluir `method`, `path` e `ip` no log de panic recovered, além do `stack` e `err` já existentes.

#### Scenario: Panic em request HTTP
- **WHEN** um panic ocorre durante o processamento de uma request HTTP
- **THEN** o log de error DEVE conter os campos `err`, `stack`, `method`, `path`, `ip` e `component=http`

### Requirement: Correção de nível de log para AppStateSyncError
O evento `AppStateSyncError` DEVE ser logado como WARN, não como DEBUG, pois representa uma condição de erro.

#### Scenario: Recebimento de AppStateSyncError
- **WHEN** o evento `AppStateSyncError` é recebido do whatsmeow
- **THEN** o log DEVE ser emitido com nível WARN (não DEBUG)
