## 1. Inbound — parser.go

- [ ] 1.1 Adicionar `"ptvMessage": "video"` ao mapa `mediaTypeMap` em `internal/integrations/chatwoot/parser.go`
- [ ] 1.2 Criar funções `extractListResponseText(msg map[string]any) string` e `extractButtonResponseText(msg map[string]any) string` em `parser.go` (reaproveitando lógica de `wa_messages.go`)
- [ ] 1.3 Adicionar `"ptvMessage"` aos slices de detecção de tipo em `parser.go` (`msgTypeKeys` → `{"ptvMessage", "video"}`)

## 2. Inbound — inbox_cloud.go: unwrap e novos tipos

- [ ] 2.1 Criar função `unwrapMessage(msg map[string]any) map[string]any` em `inbox_cloud.go` — desembrulha iterativamente (máx 5 níveis) os wrappers: `ephemeralMessage`, `viewOnceMessage`, `viewOnceMessageV2`, `viewOnceMessageV2Extension`, `interactiveMessage`
- [ ] 2.2 Chamar `unwrapMessage` no `HandleMessage` logo após `msg := data.Message` (antes de qualquer detecção de tipo)
- [ ] 2.3 Adicionar bloco `else if contactsMsg := getMapField(msg, "contactsArrayMessage"); contactsMsg != nil` no `HandleMessage` — iterar sobre `contacts[]`, parsear cada vCard com `parseVCardToCloudContacts`, consolidar e chamar `buildCloudContactMessage`
- [ ] 2.4 Adicionar bloco `else if listResp := getMapField(msg, "listResponseMessage"); listResp != nil` — extrair texto com `extractListResponseText`, chamar `buildCloudTextMessage`
- [ ] 2.5 Adicionar bloco `else if btnsResp := getMapField(msg, "buttonsResponseMessage"); btnsResp != nil` — extrair texto com `extractButtonResponseText`, chamar `buildCloudTextMessage`
- [ ] 2.6 Adicionar bloco `else if tmplReply := getMapField(msg, "templateButtonReplyMessage"); tmplReply != nil` — extrair `selectedDisplayText` ou `selectedId`, formatar `[Botão] X`, chamar `buildCloudTextMessage`
- [ ] 2.7 Adicionar bloco `else if nativeFlow := getMapField(msg, "nativeFlowMessage"); nativeFlow != nil` — parsear `buttons[0].buttonParamsJson`, extrair `payment_settings[0]` de tipo PIX, formatar texto e chamar `buildCloudTextMessage`

## 3. Outbound — dto/cloud_api.go: tipos para interactive

- [ ] 3.1 Adicionar struct `CloudAPIInteractive` em `internal/dto/cloud_api.go` com campos: `Type string`, `Body *CloudAPIInteractiveBody`, `Action *CloudAPIInteractiveAction`
- [ ] 3.2 Adicionar structs auxiliares: `CloudAPIInteractiveBody{Text string}`, `CloudAPIInteractiveAction{Buttons []CloudAPIInteractiveButton, Button string, Sections []CloudAPIInteractiveSection}`, `CloudAPIInteractiveButton{Type string, Reply *CloudAPIInteractiveReply}`, `CloudAPIInteractiveReply{ID, Title string}`, `CloudAPIInteractiveSection{Title string, Rows []CloudAPIInteractiveSectionRow}`, `CloudAPIInteractiveSectionRow{ID, Title, Description string}`
- [ ] 3.3 Adicionar campo `Interactive *CloudAPIInteractive \`json:"interactive,omitempty"\`` em `CloudAPIMessageReq`

## 4. Outbound — cloud_api.go: handler interactive + sticker

- [ ] 4.1 Criar função `handleInteractiveSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error` em `internal/handler/cloud_api.go`
  - Se `interactive.type == "button"`: mapear `buttons[].reply` para `[]dto.ButtonItem` e chamar `messageSvc.SendButton()`
  - Se `interactive.type == "list"`: mapear `sections` para `[]dto.ListSection` e chamar `messageSvc.SendList()`
  - Caso contrário: retornar 400 `"Unsupported interactive type"`
- [ ] 4.2 Adicionar `case "interactive": return h.handleInteractiveSend(c, cfg, req, to)` no switch do `SendMessage`
- [ ] 4.3 Adicionar `case "sticker": return h.handleMediaSend(c, cfg, req, to, "document")` no switch do `SendMessage`

## 5. Testes — parser e inbox_cloud

- [ ] 5.1 Adicionar testes em `parser_test.go` para `ptvMessage` em `extractMediaInfo` e `detectMessageType`
- [ ] 5.2 Criar testes unitários para `unwrapMessage` em `inbox_cloud_test.go` — cenários: ephemeral wrapping texto, viewOnce wrapping imagem, sem wrapper (passthrough), wrapper sem inner message
- [ ] 5.3 Criar testes para `contactsArrayMessage` em `inbox_cloud_test.go` — 1 contato, 2+ contatos, vCard inválido
- [ ] 5.4 Criar testes para respostas interativas (`listResponseMessage`, `buttonsResponseMessage`, `templateButtonReplyMessage`) em `inbox_cloud_test.go`
- [ ] 5.5 Criar testes para `nativeFlowMessage` PIX em `inbox_cloud_test.go` — pix_static_code, pix_dynamic_code, JSON inválido

## 6. Testes — cloud_api handler

- [ ] 6.1 Adicionar testes em `cloud_api_test.go` para `case "interactive"` com `type = "button"` (2 botões)
- [ ] 6.2 Adicionar testes em `cloud_api_test.go` para `case "interactive"` com `type = "list"` (1 seção, 2 linhas)
- [ ] 6.3 Adicionar testes em `cloud_api_test.go` para `case "interactive"` com tipo desconhecido → 400
- [ ] 6.4 Adicionar testes em `cloud_api_test.go` para `case "sticker"` → comportamento de document
