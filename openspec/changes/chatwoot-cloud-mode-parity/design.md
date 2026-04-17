## Context

O modo Cloud do Chatwoot usa a emulação da Graph API da Meta implementada em `internal/handler/cloud_api.go`. No fluxo inbound (WA → Chatwoot), `internal/integrations/chatwoot/inbox_cloud.go` converte eventos whatsmeow para payloads Cloud API e os entrega via POST para o Chatwoot. No fluxo outbound (Chatwoot → WA), o Chatwoot chama o endpoint `POST /:version/:phone/messages` com payloads Cloud API e o handler os converte para chamadas do `MessageService`.

**Estado atual:**
- Inbound cobre: texto, mídia (image/video/audio/document/sticker), localização, contato único, reação
- Outbound cobre: text, image, video, audio, document, location, reaction, contacts, template
- `ptvMessage` ausente do `mediaTypeMap` → descartado
- Wrapper types (`ephemeralMessage`, `viewOnceMessage`, etc.) não são desembrulhados → descartados
- Respostas interativas (`listResponseMessage`, `buttonsResponseMessage`, `templateButtonReplyMessage`) → descartadas
- PIX via `nativeFlowMessage` → descartado
- `contactsArrayMessage` → descartado (apenas singular suportado)
- Outbound `interactive` e `sticker` → retornam 400

## Goals / Non-Goals

**Goals:**
- Nenhum evento whatsmeow inbound deve ser descartado silenciosamente por falta de handler
- Respostas interativas do usuário chegam ao Chatwoot como texto legível
- Agentes podem enviar botões (`button`) e listas (`list`) pelo Chatwoot
- `ptvMessage` (vídeo nota) é entregue como vídeo
- Múltiplos contatos num mesmo envio chegam como Cloud API `contacts`
- Sticker pode ser enviado pelo Chatwoot via Cloud API (fallback document)

**Non-Goals:**
- Modo API (`inbox_api.go`) — sem alterações
- Envio de `interactive` pelo agente (criação proativa de botões/listas via wzap API) — já existente via `SendButton`/`SendList`; aqui apenas o path Cloud API
- `pollCreationMessage` — sem equivalente no Cloud API format
- Tratamento de `lottieStickerMessage` como tipo distinto de `stickerMessage`

## Decisions

### D1 — Função `unwrapMessage` isolada em `inbox_cloud.go`

**Decisão:** Criar função pura `unwrapMessage(msg map[string]any) map[string]any` que desembrulha iterativamente (não recursivamente) os wrapper types conhecidos.

**Alternativa considerada:** Recursão direta no `HandleMessage` (como faz o unoapi em TypeScript). Rejeitado: mais difícil de testar unitariamente e pode causar loop infinito em payloads malformados.

**Wrapper types cobertos:** `ephemeralMessage`, `viewOnceMessage`, `viewOnceMessageV2`, `viewOnceMessageV2Extension`, `interactiveMessage`

**Implementação:**
```go
func unwrapMessage(msg map[string]any) map[string]any {
    wrappers := []string{
        "ephemeralMessage", "viewOnceMessage",
        "viewOnceMessageV2", "viewOnceMessageV2Extension",
        "interactiveMessage",
    }
    for _, key := range wrappers {
        if inner := getMapField(msg, key); inner != nil {
            if nested := getMapField(inner, "message"); nested != nil {
                return nested
            }
            return inner
        }
    }
    return msg
}
```
Chamada no `HandleMessage` logo após `msg := data.Message`.

---

### D2 — Respostas interativas convertidas para texto (não Cloud API `button`/`interactive`)

**Decisão:** Converter `listResponseMessage`, `buttonsResponseMessage` e `templateButtonReplyMessage` para texto formatado (`[Lista] Opção`, `[Botão] Opção`) e enviar como `buildCloudTextMessage`.

**Alternativa considerada:** Enviar como Cloud API `button` ou `interactive` type. Rejeitado: Chatwoot Cloud mode não renderiza esses tipos em conversas inbound de forma confiável; texto garante visibilidade.

**Extração:** Reutilizar `extractButtonText` e `extractListText` já existentes em `wa_messages.go` — mover ou duplicar em `parser.go` para evitar dependência cruzada.

---

### D3 — `ptvMessage` → `mediaTypeMap` em `parser.go`

**Decisão:** Adicionar `"ptvMessage": "video"` no mapa `mediaTypeMap`. Nenhuma lógica nova necessária — o fluxo de mídia existente cuida do download, upload e construção do Cloud API payload.

---

### D4 — `contactsArrayMessage` → Cloud API `contacts` array

**Decisão:** Adicionar `else if` no `HandleMessage` para `contactsArrayMessage`. Iterar sobre `contacts[]`, parsear cada vCard com a função `parseVCardToCloudContacts` existente e consolidar num único `buildCloudContactMessage`.

---

### D5 — PIX via `nativeFlowMessage` → texto formatado

**Decisão:** Após o `unwrapMessage` desembrulhar `interactiveMessage`, verificar se a mensagem interna é `nativeFlowMessage`. Se for, extrair `payment_settings[0]` do `buttonParamsJson` do primeiro botão e formatar como:
```
*<merchant_name>*
Chave PIX tipo *<key_type>*: <key>
```
Enviar como `buildCloudTextMessage`. Esta é a mesma abordagem do unoapi-cloud.

**Nota:** O `unwrapMessage` (D1) desembrulha `interactiveMessage` → retorna o inner message que contém `nativeFlowMessage`. O handler então detecta `nativeFlowMessage` como um `else if`.

---

### D6 — Outbound `interactive`: novo campo no DTO + novo handler

**Decisão:** 
1. Adicionar campo `Interactive *CloudAPIInteractive` em `dto.CloudAPIMessageReq`
2. Criar tipos `CloudAPIInteractive`, `CloudAPIInteractiveAction`, `CloudAPIInteractiveButton`, `CloudAPIInteractiveSection` em `dto/cloud_api.go`
3. Adicionar `case "interactive"` em `cloud_api.go::SendMessage` que:
   - Se `action.type == "button"` → chama `messageSvc.SendButton()` com `dto.SendButtonReq`
   - Se `action.type == "list"` → chama `messageSvc.SendList()` com `dto.SendListReq`
4. Adicionar `case "sticker"` que delega para `handleMediaSend(..., "document")` (fallback seguro)

**Cloud API Interactive format (inbound do Chatwoot):**
```json
{
  "type": "interactive",
  "interactive": {
    "type": "button",
    "body": { "text": "Escolha:" },
    "action": {
      "buttons": [
        { "type": "reply", "reply": { "id": "1", "title": "Opção 1" } }
      ]
    }
  }
}
```

```json
{
  "type": "interactive",
  "interactive": {
    "type": "list",
    "body": { "text": "Escolha:" },
    "action": {
      "button": "Ver opções",
      "sections": [
        { "title": "Sec", "rows": [{ "id": "r1", "title": "Row 1" }] }
      ]
    }
  }
}
```

---

## Risks / Trade-offs

| Risco | Mitigação |
|---|---|
| `interactiveMessage` pode não ter inner `message` (ex: só tem `nativeFlowMessage` direto) | `unwrapMessage` retorna o inner object mesmo sem campo `message`; handler de nativeFlow verifica campo específico |
| `viewOnceMessageV2Extension` com áudio pode chegar sem directPath (stream-only) | Coberto pelo fluxo de erro de mídia existente: cai no fallback de texto/caption |
| Chatwoot pode enviar `interactive.type` com valores não cobertos (ex: `cta_url`) | `default` no switch retorna 400 com `Unsupported interactive type` |
| Múltiplos wrappers aninhados (ephemeral dentro de viewOnce) | Loop iterativo em `unwrapMessage` com máximo 5 iterações para evitar loop infinito |
| `nativeFlowMessage` com `payment_settings` vazio ou tipo desconhecido | Guard nil check em cada camada; se não for PIX conhecido, cai no fallback de texto genérico |

## Migration Plan

Sem migrações de banco de dados. Deploy sem downtime:
1. Fazer build do binário com as mudanças
2. Reiniciar o serviço wzap (graceful shutdown existente)
3. Testar manualmente: enviar mensagem efêmera, vídeo nota, lista de contatos e clicar em botão no WhatsApp
4. Rollback: deploy da versão anterior (mudanças são aditivas, nenhuma breaking change)

## Open Questions

- O Chatwoot v4+ envia `interactive` no webhook de agente para Cloud inboxes? (A validar com payload real — implementação pode ficar pronta mas inativa se o Chatwoot não suportar ainda)
