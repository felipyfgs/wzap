## ADDED Requirements

### Requirement: Endpoint de envio de mensagens simulando Graph API

O sistema DEVE expor `POST /:version/:phone/messages` que aceita payload no formato Cloud API do Meta e despacha a mensagem para o WhatsApp via whatsmeow.

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

---

### Requirement: Autenticação do endpoint Cloud API simulado

O sistema DEVE validar a identidade do Chatwoot usando Bearer token no header `Authorization`.

#### Scenario: Token válido
- **WHEN** Chatwoot envia `Authorization: Bearer <token>` e o token corresponde ao `webhook_token` da config chatwoot da sessão
- **THEN** a requisição é aceita e processada

#### Scenario: Token inválido
- **WHEN** Chatwoot envia um token que não corresponde a nenhuma sessão com aquele phone
- **THEN** retornar HTTP 401 com:
  ```json
  { "error": { "message": "Invalid access token", "type": "OAuthException", "code": 190 } }
  ```

#### Scenario: Sem header Authorization
- **WHEN** a requisição não contém header `Authorization`
- **THEN** retornar HTTP 401 com erro Cloud API format

#### Scenario: Sessão não encontrada pelo phone
- **WHEN** o `phone` da URL não corresponde a nenhuma sessão com config chatwoot cloud
- **THEN** retornar HTTP 404 com:
  ```json
  { "error": { "message": "Phone number not found", "type": "OAuthException", "code": 100 } }
  ```

---

### Requirement: Resolução de sessão por phone number

O sistema DEVE mapear o `phone` (= `phone_number_id`) da URL para a sessão wzap correspondente.

#### Scenario: Phone corresponde ao número da sessão
- **WHEN** `phone` = `"5511888888888"` e existe sessão com número `+5511888888888` e `inbox_type = "cloud"`
- **THEN** a sessão é resolvida corretamente

#### Scenario: Phone com formato variável
- **WHEN** `phone` chega como `"5511888888888"` (sem +) ou `"55 11 88888-8888"` (formatado)
- **THEN** o sistema DEVE normalizar removendo caracteres não-numéricos antes do lookup

#### Scenario: Múltiplas sessões com mesmo phone (edge case)
- **WHEN** existem 2+ sessões com o mesmo número e `inbox_type = "cloud"`
- **THEN** o sistema DEVE usar a primeira sessão encontrada e logar warning

---

### Requirement: Mark as read via Cloud API

O sistema DEVE suportar o comando de mark-as-read enviado pelo Chatwoot no formato Cloud API.

#### Scenario: Chatwoot envia mark as read
- **WHEN** Chatwoot faz POST para `/:version/:phone/messages` com:
  ```json
  {
    "messaging_product": "whatsapp",
    "status": "read",
    "message_id": "<wa_message_id>"
  }
  ```
- **THEN** o sistema DEVE chamar `messageSvc.MarkRead()` com o message_id
- **THEN** retornar HTTP 200 com `{ "success": true }`

---

### Requirement: Resposta no formato Cloud API para erros de envio

O sistema DEVE retornar erros no formato Cloud API do Meta.

#### Scenario: Falha no envio da mensagem
- **WHEN** `messageSvc.SendText()` retorna erro
- **THEN** retornar HTTP 500 com:
  ```json
  {
    "error": {
      "message": "internal server error",
      "type": "OAuthException",
      "code": 131000,
      "error_data": { "messaging_product": "whatsapp", "details": "Failed to send message" }
    }
  }
  ```

#### Scenario: Tipo de mensagem não suportado
- **WHEN** Chatwoot envia `type` desconhecido (ex: `"template"`)
- **THEN** retornar HTTP 400 com:
  ```json
  {
    "error": {
      "message": "Unsupported message type",
      "type": "OAuthException",
      "code": 131009
    }
  }
  ```

#### Scenario: Payload inválido
- **WHEN** o body da requisição não é JSON válido ou falta campos obrigatórios
- **THEN** retornar HTTP 400 com erro Cloud API format

---

### Requirement: Webhook verification endpoint

O sistema DEVE expor endpoint de verificação de webhook para Chatwoot validar a conexão.

#### Scenario: Verificação bem-sucedida
- **WHEN** Chatwoot faz `GET /:version/:phone/messages?hub.mode=subscribe&hub.verify_token=<token>&hub.challenge=<challenge>`
- **THEN** validar `verify_token` contra `webhook_token` da config chatwoot da sessão
- **THEN** retornar HTTP 200 com o `challenge` como body (plain text)

#### Scenario: Token de verificação inválido
- **WHEN** `hub.verify_token` não corresponde ao `webhook_token` configurado
- **THEN** retornar HTTP 403
