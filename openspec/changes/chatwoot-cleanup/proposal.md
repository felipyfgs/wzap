## Why

O pacote `internal/integrations/chatwoot/` cresceu para ~6.693 LOC em 29 arquivos, acumulando duplicação (inbox_api.go + inbox_cloud.go compartilham ~55 LOC de prólogo), nomenclatura inconsistente (prefixos `cw_*`, `wa_*`, `inbox_*` sem critério claro), arquivos grandes (wa_events.go 663 LOC, inbox_cloud.go 590, parser.go 591, cw_webhook.go 507), um `Service` tendendo a god object (433 LOC, 16 setters, 14+ métodos `process*`) e código morto (`buildCloudReactionMessage`, `qrcode.go`, `urlFilename` duplicado). A base está difícil de navegar antes de próximas evoluções (ex.: novas capacidades cloud, observabilidade), e tocar esse código hoje tem alto custo cognitivo.

## What Changes

- Remover código morto: `buildCloudReactionMessage` (inbox_cloud.go:323), wrapper `UnlockCloudWindow` (inbox_cloud.go:533), anchor inútil em `inbox.go:38`, arquivo `qrcode.go` (wrapper trivial de 9 LOC).
- Unificar helpers duplicados: `urlFilename` (jid.go) + `filenameFromURL` (cw_webhook.go) → manter apenas `filenameFromURL`.
- Renomear arquivos para remover prefixos redundantes: `cw_webhook→webhook_outbound`, `cw_conversation→conversation`, `cw_mapping→mapping`, `cw_backfill→backfill`, `cw_labels→labels`, `cw_bot→bot`, `wa_messages→message_types`, `wa_helpers→message_builder`.
- Extrair prólogo compartilhado dos inbox handlers para `inbox_common.go` (parse + LID + filtro + idempotência).
- Reduzir `service.go` de 433 → ~180 LOC: extrair `historyImporter` para `import.go` e `eventDispatcher` para `events.go`. Unexportar `ClearConfigCache`.
- Dividir arquivos grandes: `wa_events.go` em 3 (message_lifecycle, session, contact_group); `inbox_cloud.go` em 3 (handler, builders, transport); `parser.go` em 2 (parser + extractors); `webhook_outbound.go` em 2 (outbound + attachments).
- **Nenhuma mudança de comportamento**: refatoração puramente estrutural; contratos HTTP, webhooks, eventos NATS e APIs externas permanecem idênticos.

## Não-objetivos

- Não alterar comportamento de nenhum fluxo (inbound WA, outbound CW, webhook, import, backfill, cloud inbox).
- Não mudar schema do banco, migrações ou contratos de API.
- Não introduzir novas capacidades, endpoints ou eventos.
- Não alterar interfaces públicas do pacote (exports usados por `internal/server/router.go`, `internal/handler/session.go`, `internal/handler/cloud_api.go`).
- Não mexer em `circuit_breaker.go`, `cache.go`, `client.go`, `consumer.go`, `repo.go`, `config.go`, `tracing.go` (já bem organizados).
- Não introduzir subdiretórios: manter estrutura flat do pacote.

## Capabilities

### New Capabilities

- `chatwoot-package-hygiene`: Invariantes estruturais do pacote `internal/integrations/chatwoot/` — preservação de contrato externo, ausência de código morto, limites de tamanho de arquivo, padronização de nomenclatura. Verificável via code review e build/test automatizados.

### Modified Capabilities

- Nenhuma. Specs existentes (`cloud-inbox-inbound`, `cloud-inbox-outbound`, `cloud-inbox-media`, `live-message-filtering`, `media-retry-rate-limit`, `unknown-messages-cleanup`) permanecem válidas sem alteração — este refactor preserva o comportamento por design.

## Impact

- **Código**: `internal/integrations/chatwoot/` — 1 arquivo deletado (`qrcode.go`), 8 renomeados, 4 divididos, 1 novo arquivo compartilhado (`inbox_common.go`). Net ~−35 LOC.
- **Callers externos**: `internal/server/router.go` (wiring), `internal/handler/session.go` (acesso ao repo), `internal/handler/cloud_api.go` (integração). Nenhuma mudança de import path é necessária (mesmo pacote).
- **Testes**: `parser_test.go`, `cw_webhook_test.go`, `handler_test.go`, `service_test.go`, `testhelpers_test.go`, `cw_mapping_test.go`, `cw_backfill_test.go` — devem continuar passando sem modificações de lógica; apenas ajustes de nome de arquivo quando o teste for renomeado junto.
- **Riscos e mitigações**:
  - *Regressão silenciosa* ao extrair prólogo compartilhado → mitigar com paridade rigorosa (idempotência DB no API, cache no cloud) e rodar toda a suite de testes após cada extração.
  - *Conflito com WIP* (change `chatwoot-cloud-mode-parity` tem 817 LOC pendentes não commitadas) → commitar o WIP antes de iniciar, refatorar sobre base limpa.
  - *Histórico git perdido* em renames → usar `git mv` para preservar blame.
  - *Imports quebrados* em arquivos que movem símbolos → validar `go build ./...` e `go vet ./...` após cada passo.
- **Infra/Deploy**: nenhum. Refactor puro de código Go; sem alteração em Dockerfile, migrações, NATS streams ou configuração.
