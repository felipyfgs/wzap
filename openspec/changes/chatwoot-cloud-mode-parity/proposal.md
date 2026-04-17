## Why

O modo Cloud do Chatwoot ignora silenciosamente tipos de mensagem WhatsApp amplamente usados no Brasil — mensagens efêmeras, vídeo nota (PTV), múltiplos contatos, respostas a listas/botões e pagamentos PIX. Agentes perdem contexto crítico porque esses eventos chegam do whatsmeow mas nunca são entregues ao Chatwoot. Na direção inversa, o agente não consegue enviar mensagens interativas (botões, lista) pois o endpoint Cloud API emulado não possui o `case "interactive"`.

## What Changes

- **Inbound (`inbox_cloud.go`)**: adicionar desembrulhamento de wrapper types (`ephemeralMessage`, `viewOnceMessage`, `viewOnceMessageV2`, `viewOnceMessageV2Extension`, `interactiveMessage`) antes da detecção de tipo
- **Inbound**: suportar `contactsArrayMessage` — múltiplos contatos em uma mensagem
- **Inbound**: mapear `ptvMessage` (vídeo nota redondo) como `video` no `mediaTypeMap` de `parser.go`
- **Inbound**: converter respostas interativas (`listResponseMessage`, `buttonsResponseMessage`, `templateButtonReplyMessage`) para texto legível e enviá-las ao Chatwoot como mensagem de texto
- **Inbound**: extrair chave PIX de `nativeFlowMessage` (dentro de `interactiveMessage`) e enviar como texto formatado
- **Outbound (`cloud_api.go`)**: adicionar `case "interactive"` no `SendMessage` para suportar envio de botões e listas a partir do Chatwoot
- **Outbound**: adicionar `case "sticker"` como fallback para document

### Não-objetivos

- Não alterar o modo API (`inbox_api.go`, `wa_messages.go`) — apenas Cloud mode
- Não implementar envio de botões interativos a partir de ação do agente (futuro)
- Não tratar `pollCreationMessage` — o Chatwoot Cloud API não tem tipo equivalente
- Não alterar a lógica de status/receipts no Cloud mode

## Capabilities

### New Capabilities

_(nenhuma — todas as mudanças estendem capacidades existentes)_

### Modified Capabilities

- `cloud-inbox-inbound`: adição de wrapper type unwrapping, suporte a `contactsArrayMessage`, `ptvMessage`, respostas interativas e PIX via `nativeFlowMessage`
- `cloud-inbox-outbound`: adição dos tipos `interactive` (botões/lista) e `sticker` no endpoint Cloud API emulado

## Impact

- `internal/integrations/chatwoot/inbox_cloud.go` — lógica principal inbound
- `internal/integrations/chatwoot/parser.go` — `mediaTypeMap` (ptvMessage)
- `internal/handler/cloud_api.go` — switch do `SendMessage`
- `internal/dto/` — possível novo campo em `CloudAPIMessageReq` para interactive
- Testes: `inbox_cloud_test.go`, `parser_test.go`, `cloud_api_test.go`

## Riscos e mitigações

| Risco | Mitigação |
|---|---|
| Wrapper types aninhados (ephemeral dentro de viewOnce) | Função `unwrapMessage` itera recursivamente até encontrar tipo leaf |
| `interactiveMessage` pode não ter inner message em alguns casos (sem conteúdo legível) | Se `unwrapMessage` não encontrar tipo reconhecível, cair silenciosamente como antes |
| Chatwoot pode não suportar `interactive` como tipo de entrada no endpoint Cloud | Validar com payload real do Chatwoot; fallback para erro 400 explícito |
| `ptvMessage` sem directPath (download inviável) | Já coberto pelo fluxo de erro de mídia existente |
