## ADDED Requirements

### Requirement: Configuração do modo cloud na integração Chatwoot

O sistema DEVE permitir configurar `inbox_type` com valores `"api"` (padrão) ou `"cloud"` na integração Chatwoot de cada sessão. A coluna `inbox_type` DEVE existir na tabela `wz_chatwoot` com default `'api'`.

#### Scenario: Criar config com inbox_type cloud
- **WHEN** o operador faz `PUT /sessions/:sessionId/integrations/chatwoot` com `{"inboxType": "cloud", ...}`
- **THEN** o sistema salva `inbox_type = 'cloud'` na tabela `wz_chatwoot`
- **THEN** a resposta inclui `"inboxType": "cloud"`

#### Scenario: Criar config sem inbox_type (retrocompatível)
- **WHEN** o operador faz `PUT /sessions/:sessionId/integrations/chatwoot` sem o campo `inboxType`
- **THEN** o sistema usa o default `"api"`
- **THEN** o comportamento da integração permanece idêntico ao atual

#### Scenario: Valor inválido para inbox_type
- **WHEN** o operador envia `{"inboxType": "invalido"}`
- **THEN** o sistema retorna `400 Bad Request` com mensagem de validação

---

### Requirement: Roteamento inbound por inbox_type

Quando uma mensagem WhatsApp chega via whatsmeow, o sistema DEVE rotear o processamento de acordo com o `inbox_type` da config Chatwoot da sessão.

#### Scenario: Mensagem inbound com inbox_type api
- **WHEN** uma mensagem WA chega e `cfg.InboxType == "api"`
- **THEN** o sistema processa via fluxo existente (`handleMessage` → `client.CreateMessage` / `client.CreateMessageWithAttachment`)

#### Scenario: Mensagem inbound com inbox_type cloud
- **WHEN** uma mensagem WA chega e `cfg.InboxType == "cloud"`
- **THEN** o sistema DEVE delegar para `handleMessageCloud()`
- **THEN** o sistema NÃO DEVE chamar `client.CreateMessage` nem `client.CreateMessageWithAttachment`

---

### Requirement: Conversão de mensagem texto para formato Cloud API webhook

O sistema DEVE converter mensagens de texto do WhatsApp para o formato de webhook da Cloud API do Meta e enviar via POST para o Chatwoot.

#### Scenario: Mensagem de texto simples
- **WHEN** uma mensagem de texto chega do WhatsApp com body `"Olá"` do número `5511999999999`
- **THEN** o sistema DEVE fazer POST para `{cfg.URL}/webhooks/whatsapp` com payload:
  ```json
  {
    "object": "whatsapp_business_account",
    "entry": [{
      "id": "<session_phone>",
      "changes": [{
        "value": {
          "messaging_product": "whatsapp",
          "metadata": {
            "display_phone_number": "<session_phone>",
            "phone_number_id": "<session_phone>"
          },
          "messages": [{
            "from": "5511999999999",
            "id": "<wa_message_id>",
            "timestamp": "<unix_timestamp>",
            "type": "text",
            "text": { "body": "Olá" }
          }],
          "contacts": [{
            "profile": { "name": "<push_name>" },
            "wa_id": "5511999999999"
          }],
          "statuses": [],
          "errors": []
        },
        "field": "messages"
      }]
    }]
  }
  ```
- **THEN** o POST DEVE usar `Content-Type: application/json`

#### Scenario: Mensagem com reply-to (contexto)
- **WHEN** a mensagem WA tem `contextInfo.quotedMessageID` preenchido
- **THEN** o campo `context.message_id` DEVE estar presente no objeto `message` do payload

---

### Requirement: Conversão de mensagem de mídia para formato Cloud API webhook

O sistema DEVE converter mensagens de mídia (image, video, audio, document, sticker) para formato Cloud API, incluindo URL de download da mídia.

#### Scenario: Mensagem de imagem com caption
- **WHEN** uma mensagem de imagem chega com caption `"Foto da reunião"` e mime_type `image/jpeg`
- **THEN** o sistema DEVE:
  1. Baixar a mídia do WhatsApp via `mediaDownloader`
  2. Fazer upload para MinIO com chave `chatwoot/<sessionID>/<msgID>/<filename>`
  3. Obter URL pré-assinada via `mediaPresigner`
  4. Montar payload com `type: "image"` e campo `image.link` contendo a URL
  5. POST para `{cfg.URL}/webhooks/whatsapp`
- **THEN** o campo `image` DEVE conter: `{ "link": "<url>", "mime_type": "image/jpeg", "caption": "Foto da reunião" }`

#### Scenario: Mensagem de documento
- **WHEN** uma mensagem de documento chega com filename `relatorio.pdf` e mime_type `application/pdf`
- **THEN** o campo `document` DEVE conter: `{ "link": "<url>", "mime_type": "application/pdf", "filename": "relatorio.pdf" }`

#### Scenario: Mensagem de áudio (PTT)
- **WHEN** uma mensagem de áudio chega com mime_type `audio/ogg; codecs=opus`
- **THEN** o campo `audio` DEVE conter: `{ "link": "<url>", "mime_type": "audio/ogg; codecs=opus" }`

#### Scenario: Mensagem de vídeo
- **WHEN** uma mensagem de vídeo chega com caption `"Vídeo"` e mime_type `video/mp4`
- **THEN** o campo `video` DEVE conter: `{ "link": "<url>", "mime_type": "video/mp4", "caption": "Vídeo" }`

#### Scenario: Mensagem de sticker
- **WHEN** uma mensagem de sticker chega com mime_type `image/webp`
- **THEN** o campo `sticker` DEVE conter: `{ "link": "<url>", "mime_type": "image/webp" }`

#### Scenario: Falha no download de mídia
- **WHEN** o download da mídia falha
- **THEN** o sistema DEVE logar warning com `component: "chatwoot"` e NÃO enviar o webhook
- **THEN** a falha NÃO DEVE impedir o processamento de outras mensagens

#### Scenario: MinIO indisponível
- **WHEN** o upload para MinIO falha
- **THEN** o sistema DEVE logar warning e tentar enviar o webhook apenas com o texto/caption (sem mídia)

---

### Requirement: Conversão de mensagens especiais para formato Cloud API

O sistema DEVE converter mensagens de localização, contato e reação para o formato Cloud API.

#### Scenario: Mensagem de localização
- **WHEN** uma mensagem de localização chega com latitude `-27.59`, longitude `-48.55`, name `"Escritório"`
- **THEN** o payload DEVE conter `type: "location"` com `location: { latitude: -27.59, longitude: -48.55, name: "Escritório" }`

#### Scenario: Mensagem de contato
- **WHEN** uma mensagem de contato (vCard) chega
- **THEN** o payload DEVE conter `type: "contacts"` com array `contacts` formatado conforme Cloud API

#### Scenario: Reação a mensagem
- **WHEN** uma reação chega com emoji `"👍"` referenciando message_id `"wamid.xxx"`
- **THEN** o payload DEVE conter `type: "reaction"` com `reaction: { message_id: "wamid.xxx", emoji: "👍" }`

---

### Requirement: Tratamento de erros no POST para Chatwoot

O sistema DEVE tratar erros de rede e respostas HTTP de erro ao enviar webhooks para o Chatwoot.

#### Scenario: Chatwoot retorna 200
- **WHEN** o POST para `/webhooks/whatsapp` retorna HTTP 200
- **THEN** o sistema considera a entrega bem-sucedida

#### Scenario: Chatwoot retorna 5xx
- **WHEN** o POST retorna HTTP 500+
- **THEN** o sistema DEVE logar warning com status code e body
- **THEN** se NATS está disponível, o evento DEVE ser re-enfileirado para retry (via backoff existente)

#### Scenario: Timeout na conexão com Chatwoot
- **WHEN** o POST excede o timeout configurado (`timeoutTextSeconds` ou `timeoutMediaSeconds`)
- **THEN** o sistema DEVE logar warning e re-enfileirar se possível

#### Scenario: Sessão desconectada
- **WHEN** a integração está habilitada mas a sessão não está conectada
- **THEN** o processamento de eventos continua normalmente (eventos podem chegar de histórico ou reconexão)
