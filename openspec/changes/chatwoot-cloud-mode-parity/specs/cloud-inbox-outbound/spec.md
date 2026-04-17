## ADDED Requirements

### Requirement: Envio de mensagem interativa (botões ou lista) via Cloud API emulado

O sistema DEVE suportar o tipo `interactive` no endpoint `POST /:version/:phone/messages`, convertendo o payload Cloud API para o formato whatsmeow correspondente (`SendButton` ou `SendList`).

O campo `interactive` do payload DEVE seguir o formato Meta Cloud API:
- `interactive.type = "button"` → botões de resposta rápida
- `interactive.type = "list"` → lista com seções e linhas

#### Scenario: Envio de botões de resposta rápida (button)
- **WHEN** Chatwoot faz `POST /:version/:phone/messages` com:
  ```json
  {
    "messaging_product": "whatsapp",
    "to": "5511999999999",
    "type": "interactive",
    "interactive": {
      "type": "button",
      "body": { "text": "Escolha uma opção:" },
      "action": {
        "buttons": [
          { "type": "reply", "reply": { "id": "1", "title": "Opção 1" } },
          { "type": "reply", "reply": { "id": "2", "title": "Opção 2" } }
        ]
      }
    }
  }
  ```
- **THEN** o sistema DEVE chamar `messageSvc.SendButton()` com:
  - `phone`: JID do destinatário
  - `body`: `"Escolha uma opção:"`
  - `buttons`: `[{id:"1", text:"Opção 1"}, {id:"2", text:"Opção 2"}]`
- **THEN** retornar HTTP 200 com resposta Cloud API padrão

#### Scenario: Envio de lista (list)
- **WHEN** Chatwoot faz POST com:
  ```json
  {
    "type": "interactive",
    "interactive": {
      "type": "list",
      "body": { "text": "Selecione um departamento:" },
      "action": {
        "button": "Ver opções",
        "sections": [
          {
            "title": "Atendimento",
            "rows": [
              { "id": "sup", "title": "Suporte", "description": "Problemas técnicos" }
            ]
          }
        ]
      }
    }
  }
  ```
- **THEN** o sistema DEVE chamar `messageSvc.SendList()` com:
  - `body`: `"Selecione um departamento:"`
  - `buttonText`: `"Ver opções"`
  - `sections`: seção `"Atendimento"` com row `{id:"sup", title:"Suporte", description:"Problemas técnicos"}`
- **THEN** retornar HTTP 200 com resposta Cloud API padrão

#### Scenario: interactive.type desconhecido
- **WHEN** Chatwoot envia `interactive.type = "cta_url"` (tipo não suportado)
- **THEN** o sistema DEVE retornar HTTP 400 com:
  ```json
  { "error": { "message": "Unsupported interactive type", "type": "OAuthException", "code": 131009 } }
  ```

#### Scenario: Campo interactive ausente no payload
- **WHEN** `type = "interactive"` mas o campo `interactive` está ausente ou nulo
- **THEN** o sistema DEVE retornar HTTP 400 com erro `"Missing 'interactive' field"`

#### Scenario: Botões com array vazio
- **WHEN** `interactive.action.buttons` é um array vazio
- **THEN** o sistema DEVE retornar HTTP 400 com erro de validação

#### Scenario: Lista sem seções
- **WHEN** `interactive.type = "list"` mas `interactive.action.sections` está vazio ou ausente
- **THEN** o sistema DEVE retornar HTTP 400 com erro de validação

---

### Requirement: Envio de sticker via Cloud API emulado

O sistema DEVE suportar o tipo `sticker` no endpoint `POST /:version/:phone/messages`, tratando-o como envio de documento (fallback).

#### Scenario: Envio de sticker com link
- **WHEN** Chatwoot faz POST com `type: "sticker"` e `sticker.link` contendo URL de arquivo `.webp`
- **THEN** o sistema DEVE baixar o arquivo e enviá-lo como documento para o WhatsApp
- **THEN** retornar HTTP 200 com resposta Cloud API padrão

#### Scenario: Campo sticker ausente
- **WHEN** `type = "sticker"` mas o campo `sticker` está ausente
- **THEN** o sistema DEVE retornar HTTP 400 com erro `"Missing 'sticker' field"`

---

## MODIFIED Requirements

### Requirement: Endpoint de envio de mensagens simulando Graph API

O sistema DEVE expor `POST /:version/:phone/messages` que aceita payload no formato Cloud API do Meta e despacha a mensagem para o WhatsApp via whatsmeow, **incluindo os novos tipos `interactive` e `sticker`**.

#### Scenario: Envio de mensagem de texto
- **WHEN** Chatwoot faz `POST /v20.0/5511888888888/messages` com:
  ```json
  {
    "messaging_product": "whatsapp",
    "recipient_type": "individual",
    "to": "5511999999999",
    "type": "text",
    "text": { "body": "Olá, como posso ajudar?" }
  }
  ```
- **THEN** o sistema DEVE:
  1. Resolver a sessão pelo `phone` da URL (lookup por número)
  2. Validar Bearer token contra `webhook_token` da config chatwoot
  3. Chamar `messageSvc.SendText()` com `to: "5511999999999@s.whatsapp.net"` e body `"Olá, como posso ajudar?"`
- **THEN** retornar HTTP 200 com:
  ```json
  {
    "messaging_product": "whatsapp",
    "contacts": [{ "input": "5511999999999", "wa_id": "5511999999999" }],
    "messages": [{ "id": "<wa_message_id>" }]
  }
  ```

#### Scenario: Envio de imagem com URL
- **WHEN** Chatwoot faz POST com `type: "image"` e campo `image.link` contendo URL da imagem
- **THEN** o sistema DEVE chamar `messageSvc.SendImage()` com a URL e caption (se presente)
- **THEN** retornar HTTP 200 com resposta Cloud API

#### Scenario: Envio de documento
- **WHEN** Chatwoot faz POST com `type: "document"` e campo `document.link`
- **THEN** o sistema DEVE chamar `messageSvc.SendDocument()` com URL, filename e caption

#### Scenario: Envio de vídeo
- **WHEN** Chatwoot faz POST com `type: "video"` e campo `video.link`
- **THEN** o sistema DEVE chamar `messageSvc.SendVideo()` com URL e caption

#### Scenario: Envio de áudio
- **WHEN** Chatwoot faz POST com `type: "audio"` e campo `audio.link`
- **THEN** o sistema DEVE chamar `messageSvc.SendAudio()` com a URL

#### Scenario: Envio de localização
- **WHEN** Chatwoot faz POST com `type: "location"` e campos latitude/longitude
- **THEN** o sistema DEVE chamar `messageSvc.SendLocation()`

#### Scenario: Envio de contato
- **WHEN** Chatwoot faz POST com `type: "contacts"` e array de contatos
- **THEN** o sistema DEVE chamar `messageSvc.SendContact()` com os dados do vCard

#### Scenario: Envio de reação
- **WHEN** Chatwoot faz POST com `type: "reaction"` e campos `message_id` + `emoji`
- **THEN** o sistema DEVE chamar o endpoint de reação do whatsmeow

#### Scenario: Envio de botões interativos
- **WHEN** Chatwoot faz POST com `type: "interactive"` e `interactive.type = "button"`
- **THEN** o sistema DEVE chamar `messageSvc.SendButton()` (ver Requirement: Envio de mensagem interativa)

#### Scenario: Envio de lista interativa
- **WHEN** Chatwoot faz POST com `type: "interactive"` e `interactive.type = "list"`
- **THEN** o sistema DEVE chamar `messageSvc.SendList()` (ver Requirement: Envio de mensagem interativa)

#### Scenario: Envio de sticker
- **WHEN** Chatwoot faz POST com `type: "sticker"` e campo `sticker.link`
- **THEN** o sistema DEVE enviar como documento (ver Requirement: Envio de sticker)

#### Scenario: Tipo de mensagem não suportado
- **WHEN** Chatwoot envia `type` desconhecido (ex: `"order"`)
- **THEN** retornar HTTP 400 com:
  ```json
  { "error": { "message": "Unsupported message type", "type": "OAuthException", "code": 131009 } }
  ```
