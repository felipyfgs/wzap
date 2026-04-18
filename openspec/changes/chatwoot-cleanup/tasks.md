## 1. Pré-requisitos

- [x] 1.1 Confirmar que o WIP de `chatwoot-cloud-mode-parity` está commitado (git status limpo no pacote `internal/integrations/chatwoot/`)
- [x] 1.2 Rodar baseline: `go build ./...`, `go vet ./...`, `go test ./internal/integrations/chatwoot/...` — todos devem passar antes de iniciar
- [x] 1.3 Criar branch `refactor/chatwoot-cleanup` a partir de `main`

## 2. Passo 1 — Remover código morto e helpers duplicados (baixo risco, −35 LOC)

- [x] 2.1 Deletar função `buildCloudReactionMessage` (inbox_cloud.go:323-334)
- [x] 2.2 Deletar wrapper `UnlockCloudWindow` (inbox_cloud.go:533-535); renomear `unlockCloudWindow` privado → `UnlockCloudWindow` público
- [x] 2.3 Inlinar `qrcode.Encode(...)` no único caller em `processQR` (wa_events.go:~413) e deletar arquivo `qrcode.go`
- [x] 2.4 Remover anchor `var _ model.EventType = ""` em inbox.go:38
- [x] 2.5 Remover função duplicada `urlFilename` (jid.go:109-113); atualizar caller em `cw_conversation.go:~135` para usar `filenameFromURL` (cw_webhook.go)
- [x] 2.6 Unexportar `ClearConfigCache` → `clearConfigCache` (service.go:133); atualizar caller em `cw_conversation.go:~323`
- [x] 2.7 `go build ./... && go vet ./... && go test ./internal/integrations/chatwoot/...` — tudo passando
- [x] 2.8 Commit: `refactor(chatwoot): remove dead code and duplicated helpers`

## 3. Passo 2 — Renomear arquivos (zero mudança de conteúdo)

- [x] 3.1 `git mv internal/integrations/chatwoot/cw_webhook.go internal/integrations/chatwoot/webhook_outbound.go`
- [x] 3.2 `git mv internal/integrations/chatwoot/cw_webhook_test.go internal/integrations/chatwoot/webhook_outbound_test.go`
- [x] 3.3 `git mv internal/integrations/chatwoot/cw_conversation.go internal/integrations/chatwoot/conversation.go`
- [x] 3.4 `git mv internal/integrations/chatwoot/cw_conversation_test.go internal/integrations/chatwoot/conversation_test.go`
- [x] 3.5 `git mv internal/integrations/chatwoot/cw_mapping.go internal/integrations/chatwoot/mapping.go`
- [x] 3.6 `git mv internal/integrations/chatwoot/cw_mapping_test.go internal/integrations/chatwoot/mapping_test.go`
- [x] 3.7 `git mv internal/integrations/chatwoot/cw_backfill.go internal/integrations/chatwoot/backfill.go`
- [x] 3.8 `git mv internal/integrations/chatwoot/cw_backfill_test.go internal/integrations/chatwoot/backfill_test.go`
- [x] 3.9 `git mv internal/integrations/chatwoot/cw_labels.go internal/integrations/chatwoot/labels.go`
- [x] 3.10 `git mv internal/integrations/chatwoot/cw_labels_test.go internal/integrations/chatwoot/labels_test.go`
- [x] 3.11 `git mv internal/integrations/chatwoot/cw_bot.go internal/integrations/chatwoot/bot.go`
- [x] 3.12 `git mv internal/integrations/chatwoot/wa_messages.go internal/integrations/chatwoot/message_types.go`
- [x] 3.13 `git mv internal/integrations/chatwoot/wa_messages_test.go internal/integrations/chatwoot/message_types_test.go`
- [x] 3.14 `git mv internal/integrations/chatwoot/wa_helpers.go internal/integrations/chatwoot/message_builder.go`
- [x] 3.15 `go build ./... && go vet ./... && go test ./internal/integrations/chatwoot/...` — tudo passando
- [x] 3.16 Commit: `refactor(chatwoot): rename files to drop redundant cw_/wa_ prefixes`

## 4. Passo 3 — Extrair prólogo compartilhado dos inbox handlers

- [x] 4.1 Criar `internal/integrations/chatwoot/inbox_common.go` com struct `inboxPrologueOpts` (campo `checkDBIdempotency bool`) e método `(s *Service) inboxPrologue(ctx, cfg, payload, opts) (*waMessagePayload, string, string, bool, error)` retornando `(data, chatJID, sourceID, skip, err)`
- [x] 4.2 Implementar no helper: parse → resolve LID → valida @lid → aplica `shouldIgnoreJID` → verifica cache idempotente → (se `checkDBIdempotency`) verifica `msgRepo.FindByWAID`
- [x] 4.3 Refatorar `apiInboxHandler.HandleMessage` em `inbox_api.go` para chamar `inboxPrologue(..., opts{checkDBIdempotency: true})` e remover código duplicado do topo
- [x] 4.4 Refatorar `cloudInboxHandler.HandleMessage` em `inbox_cloud.go` para chamar `inboxPrologue(..., opts{checkDBIdempotency: false})` e remover código duplicado do topo
- [x] 4.5 Criar `internal/integrations/chatwoot/inbox_common_test.go` com teste de paridade: mesmo payload deve produzir `skip` equivalente para ambos os modos nas etapas compartilhadas
- [x] 4.6 `go build ./... && go vet ./... && go test ./internal/integrations/chatwoot/...` — tudo passando
- [x] 4.7 Commit: `refactor(chatwoot): extract shared inbox prologue to inbox_common.go`

## 5. Passo 4 — Extrair `import.go` e `events.go` de `service.go`

- [x] 5.1 Criar `internal/integrations/chatwoot/import.go` movendo funções de service.go: `ImportHistoryAsync`, `importHistory`, `importSingleMessage`, `importMediaMessage`, `importPeriodToDays` (mantêm-se como métodos em `*Service`)
- [x] 5.2 Criar `internal/integrations/chatwoot/events.go` com `type eventDispatcher struct { svc *Service }` e método `Handle(ctx, cfg, event, payload) error` contendo o switch de `processInboundEvent` (service.go:~180-217)
- [x] 5.3 Em `service.go`, refatorar `processInboundEvent` para delegar a um `eventDispatcher` instanciado lazy (ou campo do struct `Service`)
- [x] 5.4 Verificar que `service.go` ficou entre 150-220 LOC (`wc -l`) — 187 LOC
- [x] 5.5 `go build ./... && go vet ./... && go test ./internal/integrations/chatwoot/...` — tudo passando
- [x] 5.6 Commit: `refactor(chatwoot): split service.go into import.go and events.go`

## 6. Passo 5 — Dividir `wa_events.go` (663 LOC) em três arquivos

- [x] 6.1 Criar `internal/integrations/chatwoot/events_message_lifecycle.go` movendo: `processReceipt`, `processDelete`, `processRevoke`, `processEdit`, `processEditCloud`, `waitForCWRef` (345 LOC)
- [x] 6.2 Criar `internal/integrations/chatwoot/events_session.go` movendo: `processConnected`, `processDisconnected`, `processQR` (93 LOC)
- [x] 6.3 Criar `internal/integrations/chatwoot/events_contact_group.go` movendo: `processContact`, `processPushName`, `processPicture`, `processGroupInfo`, `processHistorySync` (244 LOC)
- [x] 6.4 Deletar `wa_events.go` original
- [x] 6.5 Renomeado `wa_events_test.go` → `events_test.go` (um único arquivo cobrindo os três novos)
- [x] 6.6 `go build ./... && go vet ./... && go test ./internal/integrations/chatwoot/...` — tudo passando
- [x] 6.7 Commit: `refactor(chatwoot): split wa_events.go into three lifecycle files`

## 7. Passo 6 — Dividir `inbox_cloud.go` (590 LOC) em três arquivos

- [x] 7.1 Em `inbox_cloud.go` (157 LOC): manter struct `cloudInboxHandler` e método `HandleMessage`
- [x] 7.2 Criar `internal/integrations/chatwoot/cloud_builders.go` (219 LOC): tipos de envelope cloud + builders + `parseVCardToCloudContacts`, `cloudMediaType`
- [x] 7.3 Criar `internal/integrations/chatwoot/cloud_transport.go` (195 LOC): `postToChatwootCloud`, `uploadCloudMedia`, `uploadRawMedia`, `getMediaUploader`, `UnlockCloudWindow` e helpers HTTP
- [x] 7.4 `go build ./... && go vet ./... && go test ./internal/integrations/chatwoot/...` — tudo passando
- [x] 7.5 Commit: `refactor(chatwoot): split inbox_cloud.go into handler/builders/transport`

## 8. Passo 7 — Dividir `parser.go` (591 LOC) em dois arquivos

- [x] 8.1 Em `parser.go` (136 LOC): tipos de payload, `flexTimestamp`, `parseEnvelopeData`, `parseMessagePayload`, `parseReceiptPayload`, `parseDeletePayload`, helpers (`getStringField`, `getFloatField`, `getMapField`)
- [x] 8.2 Criar `internal/integrations/chatwoot/extractors.go` (352 LOC): `detectMessageType`, `extractText`, `formatLocation`, `findNestedContextInfo`, `extractStanzaID`, `extractQuoteText`, `extractLocationFromText`, `extractMediaInfo`, `mediaTypeMap`. Helpers de VCard/string foram movidos para `vcard.go` (115 LOC) para manter extractors < 450 LOC.
- [x] 8.3 `go build ./... && go vet ./... && go test ./internal/integrations/chatwoot/...` — tudo passando
- [x] 8.4 Commit: `refactor(chatwoot): split parser.go into parser + extractors`

## 9. Passo 8 — Dividir `webhook_outbound.go` (507 LOC) em dois arquivos

- [x] 9.1 Em `webhook_outbound.go` (302 LOC): `HandleIncomingWebhook`, `syncCloudMessageRef`, `isOutboundDuplicate`, `processOutgoingMessage`, `processMessageEdited`, `processMessageUpdated`, `processStatusChanged`, const `maxMediaBytes`
- [x] 9.2 Criar `internal/integrations/chatwoot/webhook_attachments.go` (217 LOC): `sendAttachment`, `sendVCardToWhatsApp`, `resolveOutboundReply`, `signContent`, `markReadIfEnabled`, `sendErrorToAgent`, `rewriteAttachmentURL`, `filenameFromURL`
- [x] 9.3 `go build ./... && go vet ./... && go test ./internal/integrations/chatwoot/...` — tudo passando
- [x] 9.4 Commit: `refactor(chatwoot): split webhook_outbound.go into outbound + attachments`

## 10. Verificação final

- [x] 10.1 Todos os arquivos tocados por este refactor estão < 450 LOC. `client.go` (464) e `message_types.go` (442) permanecem acima/próximos do limite mas estão explicitamente fora do escopo (non-goal do design: "Reescrita de módulos já bem estruturados").
- [x] 10.2 `ls internal/integrations/chatwoot/cw_*.go internal/integrations/chatwoot/wa_*.go` — saída vazia
- [x] 10.3 `find internal/integrations/chatwoot -mindepth 1 -type d` — saída vazia (pacote flat)
- [x] 10.4 `grep -rn "buildCloudReactionMessage\|urlFilename" internal/integrations/chatwoot/` — saída vazia
- [x] 10.5 `wc -l internal/integrations/chatwoot/service.go` — 187 linhas (dentro de 150-220)
- [x] 10.6 `go build ./... && go vet ./... && go test ./...` em todo o projeto — tudo passando
- [ ] 10.7 Abrir PR `refactor/chatwoot-cleanup → main` com descrição resumindo os 8 passos e delta de LOC (deixado para o usuário)
- [ ] 10.8 Arquivar a change no openspec após merge: `openspec archive chatwoot-cleanup` (deixado para o usuário)
