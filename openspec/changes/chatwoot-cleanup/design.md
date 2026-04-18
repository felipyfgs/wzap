## Context

O pacote `internal/integrations/chatwoot/` evoluiu organicamente ao longo de várias iterações (primeiro o modo API clássico, depois o modo Cloud, mais recentemente o backfill de referências). Cada onda adicionou arquivos sem revisitar a organização global. Estado atual:

- 29 arquivos `.go` (não-teste), ~6.693 LOC.
- 4 arquivos grandes (>500 LOC): `wa_events.go`, `cw_webhook.go`, `inbox_cloud.go`, `parser.go`.
- Prefixos de arquivo inconsistentes: `cw_*` (6 arquivos, operações Chatwoot), `wa_*` (3 arquivos, conversão WhatsApp), `inbox_*` (2 arquivos, estratégia de inbox), outros sem prefixo.
- `Service` (service.go, 433 LOC) concentra 16 setters de dependência, 14+ métodos `process*`, além da lógica de `ImportHistoryAsync` (~200 LOC de import de histórico).
- Duplicação concreta: prólogo de `HandleMessage` em `inbox_api.go` e `inbox_cloud.go` (parse → LID → filtro → idempotência) ~55 LOC sobrepostos; helpers `urlFilename` e `filenameFromURL` fazem a mesma coisa em arquivos diferentes.
- Código morto confirmado via grep: `buildCloudReactionMessage` (inbox_cloud.go:323-334) nunca é chamado; wrapper `UnlockCloudWindow` (inbox_cloud.go:533-535) delega para versão privada; `qrcode.go` (9 LOC) envolve uma única chamada de lib.

**Callers externos** (verificados): `internal/server/router.go`, `internal/handler/session.go`, `internal/handler/cloud_api.go`. Todos consomem apenas símbolos exportados de alto nível (`NewService`, `NewHandler`, `NewRepository`, `NewConsumer`, setters, `OnEvent`). A refatoração deve preservar exatamente essa superfície pública.

**WIP pendente**: change `chatwoot-cloud-mode-parity` adicionou 4 arquivos novos (`cw_backfill.go`, `cw_mapping.go` e respectivos testes) e modificou 13 arquivos (~817 LOC não commitadas). Esse trabalho é funcional e deve ser preservado — a refatoração acontece *após* o commit dele.

## Goals / Non-Goals

**Goals:**

- Reduzir `service.go` de 433 → ~180 LOC via extração de `historyImporter` e `eventDispatcher`.
- Eliminar 100% da duplicação entre `inbox_api.go` e `inbox_cloud.go` no prólogo de mensagens.
- Padronizar nomenclatura de arquivos: remover prefixos `cw_*`/`wa_*` redundantes; usar nomes descritivos (ex.: `conversation.go`, `labels.go`, `webhook_outbound.go`).
- Quebrar arquivos >500 LOC em unidades coesas (<400 LOC cada).
- Remover código morto confirmado (deleções mínimas, não especulativas).
- **Preservar 100% do comportamento externo**: mesma superfície HTTP, mesmos eventos NATS, mesmos contratos com Chatwoot/WhatsApp.

**Non-Goals:**

- Mudanças de comportamento, API pública, schema de banco ou contratos externos.
- Introdução de novas capacidades, endpoints, tipos de evento ou métricas.
- Subdiretórios dentro do pacote (Go prefere pacotes flat).
- Reescrita de módulos já bem estruturados (`cache.go`, `circuit_breaker.go`, `client.go`, `consumer.go`, `config.go`, `repo.go`, `tracing.go`).
- Otimizações de performance (se encontradas, registrar como follow-up, não corrigir aqui).
- Migração para testify, mockery ou qualquer framework de teste novo.

## Decisions

### D1 — Pacote permanece flat

**Decisão:** Todos os arquivos continuam em `internal/integrations/chatwoot/` sem subdiretórios.

**Alternativa considerada:** subdiretórios `chatwoot/inbox/`, `chatwoot/webhook/`, `chatwoot/events/`.

**Rationale:** Em Go, subpacotes implicariam expor símbolos entre eles (tornando mais coisa pública do que hoje) e quebrariam os import paths em `router.go`. Com ~40 arquivos bem nomeados em um pacote, a navegação IDE já resolve descoberta. Trocar estrutura de pacotes é mudança de API interna maior do que justifica.

### D2 — Renomear em vez de mover para subpacotes

**Decisão:** Aplicar renames que remove prefixos redundantes (`cw_*`, `wa_*`) quando o contexto já está claro pelo nome. Preservar prefixo somente quando há ambiguidade (`inbox_*`, `cloud_*`, `webhook_*`).

| Atual | Novo | Motivo |
|---|---|---|
| `cw_webhook.go` | `webhook_outbound.go` | É o webhook de saída CW→WA; nome descritivo |
| `cw_conversation.go` | `conversation.go` | Pacote já é `chatwoot`; prefixo redundante |
| `cw_mapping.go` | `mapping.go` | idem |
| `cw_backfill.go` | `backfill.go` | idem |
| `cw_labels.go` | `labels.go` | idem |
| `cw_bot.go` | `bot.go` | idem |
| `wa_messages.go` | `message_types.go` | Descreve conteúdo (tipos de mensagem WA) |
| `wa_helpers.go` | `message_builder.go` | Descreve função (builders de mensagem CW) |
| `wa_events.go` | dividido em 3 | Ver D4 |

**Rationale:** Prefixos foram úteis quando `cw_*` vs `wa_*` demarcava direções, mas hoje o leitor já sabe pelo conteúdo. Usar `git mv` preserva `git blame`.

### D3 — Extrair prólogo compartilhado para `inbox_common.go`

**Decisão:** Criar helper `(s *Service).inboxPrologue(ctx, cfg, payload, opts) (*waMessagePayload, chatJID, sourceID, skip, error)` reutilizado pelos dois handlers.

**Fluxo extraído:**

```
parseMessagePayload(payload)          // 1. Desempacota envelope
  ↓
resolveLID + validação @lid           // 2. Resolve LID se chat for @lid
  ↓
shouldIgnoreJID(chatJID, ignoreJIDs)  // 3. Aplica filtro configurado
  ↓
cache.GetIdempotent("WAID:" + msgID)  // 4. Short-circuit se duplicado
  ↓
(opcional) checar msgRepo             // 5. Apenas modo API faz DB lookup extra
```

**Alternativa considerada:** manter duplicado.

**Rationale:** Os dois handlers executam rigorosamente o mesmo prólogo, apenas com um passo extra em modo API (DB lookup de mensagem já enviada). Encapsular num helper com `opts.checkDBIdempotency bool` deixa a divergência explícita e testável isoladamente.

**Risco:** paridade exige cuidado — o modo API tem um caminho `msgRepo.FindByWAID` que cloud não tem.

**Mitigação:** helper aceita flag `opts.checkDBIdempotency`; handler API passa `true`, cloud passa `false`; teste unitário de paridade compara comportamento lado-a-lado.

### D4 — Quebrar `Service` em `eventDispatcher` + `historyImporter`

**Decisão:**

- `events.go` (novo) — `type eventDispatcher struct{ svc *Service }` dona do `switch event` atualmente em `service.go:180-217` (`processInboundEvent`).
- `import.go` (novo) — move `importHistory`, `importSingleMessage`, `importMediaMessage`, `ImportHistoryAsync`, `importPeriodToDays` (~200 LOC) para arquivo dedicado. Métodos continuam em `*Service` (não criar novo tipo) — apenas reorganização de arquivo para reduzir ruído em `service.go`.

**Resultado:** `service.go` 433 → ~180 LOC (struct + construtor + setters + `OnEvent` entry + `processInboundSync`/`processOutbound` finos).

**Alternativa considerada:** Criar tipos novos para cada grupo (`botNotifier`, `contactSync`). Rejeitada — aumenta indireção sem ganho, já que os métodos são finos e orquestram deps do `Service`. Se futuramente ficarem grossos, aí refatora pra tipos próprios.

**Rationale:** A maior parte do peso em `service.go` hoje é o import histórico (200 LOC) e o dispatcher (40 LOC). Mover esses dois blocos sozinhos derruba o arquivo para tamanho saudável sem quebrar nada.

### D5 — Divisões de arquivos grandes

**`wa_events.go` (663) → 3 arquivos por ciclo de vida:**

- `events_message_lifecycle.go` (~220 LOC) — `processReceipt`, `processDelete`, `processRevoke`, `processEdit`, `processEditCloud`, `waitForCWRef`.
- `events_session.go` (~130 LOC) — `processConnected`, `processDisconnected`, `processQR` (com `qrcode.Encode` inlined).
- `events_contact_group.go` (~240 LOC) — `processContact`, `processPushName`, `processPicture`, `processGroupInfo`, `processHistorySync`.

**`inbox_cloud.go` (590) → 3 arquivos por responsabilidade:**

- `inbox_cloud.go` (~180 LOC) — struct `cloudInboxHandler` e `HandleMessage`.
- `cloud_builders.go` (~170 LOC) — tipos de envelope cloud + `buildCloud*Message` (sem o morto `buildCloudReactionMessage`).
- `cloud_transport.go` (~230 LOC) — `postToChatwootCloud`, `uploadCloudMedia`, `uploadRawMedia`, `UnlockCloudWindow` + helpers de HTTP.

**`parser.go` (591) → 2 arquivos:**

- `parser.go` (~200 LOC) — tipos de payload, `flexTimestamp`, `parseEnvelopeData`, `parseMessagePayload`, `parseReceiptPayload`, `parseDeletePayload`, helpers de acesso a map (`getStringField`, `getFloatField`, `getMapField`).
- `extractors.go` (~390 LOC) — `detectMessageType`, `extractText`, `formatLocation`, `formatVCard*`, `splitLines`, `findNestedContextInfo`, `extractStanzaID`, `extractQuoteText`, `extractLocationFromText`, `isVCardContent`, `splitVCards`, `extractVCardName`, `extractMediaInfo`.

**`webhook_outbound.go` (ex-`cw_webhook.go`, 507) → 2 arquivos:**

- `webhook_outbound.go` (~280 LOC) — `HandleIncomingWebhook`, `syncCloudMessageRef`, `isOutboundDuplicate`, `processOutgoingMessage`, `processMessageEdited`, `processMessageUpdated`, `processStatusChanged`.
- `webhook_attachments.go` (~230 LOC) — `sendAttachment`, `sendVCardToWhatsApp`, `resolveOutboundReply`, `signContent`, `markReadIfEnabled`, `sendErrorToAgent`, `rewriteAttachmentURL`, `filenameFromURL`.

### D6 — Remoções de código morto

| Item | Local | Evidência |
|---|---|---|
| `buildCloudReactionMessage` | inbox_cloud.go:323-334 | grep em todo o repo: 0 chamadas |
| Wrapper `UnlockCloudWindow` | inbox_cloud.go:533-535 | delega 1:1 para `unlockCloudWindow` privado; renomear o privado → maiúsculo e remover wrapper |
| `qrcode.go` (arquivo inteiro) | qrcode.go (9 LOC) | wrapper trivial em volta de `qrcode.Encode`; inlinar no único caller (`processQR`) |
| Anchor `var _ model.EventType = ""` | inbox.go:38 | nenhum efeito funcional |
| `urlFilename` | jid.go:109-113 | duplicado de `filenameFromURL` (cw_webhook.go:345-362); versão do webhook é mais robusta (percent-decoding) |

**Rationale:** Todos confirmados por grep antes de propor remoção. `ClearConfigCache` (service.go:133) é usada internamente por `conversation.go:323` — **não** será removida, apenas rebaixada para `clearConfigCache` (unexport).

### D7 — Ordem de execução

Passos independentes, executados em sequência com commit + `go build ./...` + `go vet ./...` + `go test ./internal/integrations/chatwoot/...` entre cada um:

1. **Dead code + helpers duplicados** (baixo risco, −35 LOC)
2. **Renames** (zero mudança de conteúdo)
3. **`inbox_common.go` extraction** (médio risco — paridade de idempotência)
4. **Split de `service.go`** (`import.go` + `events.go`)
5. **Split de `wa_events.go`**
6. **Split de `inbox_cloud.go`**
7. **Split de `parser.go`**
8. **Split de `webhook_outbound.go`**

Cada passo é um commit separado para permitir bisect fácil se algo regredir.

## Risks / Trade-offs

**[Risco] Regressão silenciosa no prólogo compartilhado (inbox_common.go)** — modo API tem DB lookup extra para idempotência que cloud não tem; se o helper unificado não respeitar essa diferença, duplicatas vazam no API ou cloud faz consulta desnecessária.
→ **Mitigação:** flag `opts.checkDBIdempotency` no helper; teste unitário de paridade rodando os dois modos lado-a-lado com mesmo payload; revisão cuidadosa do diff antes de commitar.

**[Risco] Histórico git perdido em renames** — `cw_webhook.go → webhook_outbound.go` pode aparecer como "arquivo deletado + arquivo novo" em vez de rename se o conteúdo mudar no mesmo commit.
→ **Mitigação:** executar `git mv` e commitar *só* o rename primeiro; mudanças de conteúdo (splits, deletions) vêm em commits seguintes.

**[Risco] Testes referenciam símbolos privados por posição no arquivo** — se um teste acessa algo que estava no meio de um arquivo grande, pode quebrar após split.
→ **Mitigação:** todos os testes são `package chatwoot_test` externo (conforme AGENTS.md) e dependem só da superfície pública; mesmo se houver testes internos, split preserva símbolos (só muda arquivo).

**[Risco] Conflito com WIP não commitado** (817 LOC pendentes).
→ **Mitigação:** pré-requisito de implementação é commitar WIP primeiro. Sem isso, não iniciar.

**[Risco] Divisão arbitrária de `extractors.go`** — se `extractText` for complexa e dependente de vários helpers, o split `parser.go`/`extractors.go` pode deixar um dos dois com dependências circulares.
→ **Mitigação:** Go não tem dependência circular *dentro* do mesmo pacote; a divisão é puramente organizacional, sem risco técnico.

**[Trade-off] Sem subpacotes** significa que símbolos que deveriam ser privados a um subdomínio (ex.: `buildCloudTextMessage`) continuam acessíveis de qualquer arquivo do pacote. Aceitável pelo ganho em simplicidade de refactor e navegação.

**[Trade-off] `eventDispatcher` continua acoplado ao `Service`** (tem ponteiro de volta). Não é um tipo desacoplável — só um arquivo de dispatch. Consciente: se evoluir para testar dispatch isoladamente, aí extrai interface; hoje não justifica.

## Migration Plan

Refactor interno puro — não há migração de dados, nem rollout gradual. Estratégia de deploy:

1. **Pré-commit**: WIP da change `chatwoot-cloud-mode-parity` deve estar commitado.
2. **Execução**: 8 commits sequenciais (um por passo de `tasks.md`), cada um com `go build ./...`, `go vet ./...`, `go test ./internal/integrations/chatwoot/...` passando.
3. **PR único** agregando os 8 commits com histórico preservado (não squashar — permite bisect se regressão aparecer em staging).
4. **Rollback**: `git revert` do PR inteiro; refactor puro, sem efeito em estado externo.

## Open Questions

Nenhuma. Plano é 100% baseado em leitura do código atual e evidência de `grep`/`git status`. Se algo surgir durante execução (ex.: rename gera conflito inesperado com import), ajusta dentro do passo correspondente.
