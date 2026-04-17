## ADDED Requirements

### Requirement: Desembrulhamento de wrapper types antes do processamento inbound

O sistema DEVE desembrulhar automaticamente mensagens wrapper antes de detectar o tipo da mensagem, garantindo que o conteúdo interno seja processado corretamente.

Wrapper types suportados (em ordem de verificação):
- `ephemeralMessage` → extrai `message` interno
- `viewOnceMessage` → extrai `message` interno
- `viewOnceMessageV2` → extrai `message` interno
- `viewOnceMessageV2Extension` → extrai `message` interno
- `interactiveMessage` → extrai `message` interno ou o próprio objeto

O desembrulhamento DEVE ser iterativo (até 5 níveis) para evitar loop infinito em payloads malformados. Após o desembrulhamento, o pipeline normal de detecção de tipo continua.

#### Scenario: Mensagem de texto dentro de ephemeralMessage
- **WHEN** o whatsmeow entrega payload com `message.ephemeralMessage.message.conversation = "Olá"`
- **THEN** o sistema DEVE desembrulhar e processar como mensagem de texto com body `"Olá"`
- **THEN** o webhook Cloud API entregue ao Chatwoot DEVE ter `type: "text"` com `text.body: "Olá"`

#### Scenario: Imagem dentro de viewOnceMessage
- **WHEN** o whatsmeow entrega payload com `message.viewOnceMessage.message.imageMessage = {...}`
- **THEN** o sistema DEVE desembrulhar e processar como mensagem de imagem normalmente
- **THEN** a mídia DEVE ser baixada, carregada no MinIO e entregue ao Chatwoot como `type: "image"`

#### Scenario: Áudio dentro de viewOnceMessageV2Extension
- **WHEN** o whatsmeow entrega payload com `message.viewOnceMessageV2Extension.message.audioMessage = {...}`
- **THEN** o sistema DEVE desembrulhar e processar como mensagem de áudio normalmente

#### Scenario: Wrapper sem inner message (payload inválido)
- **WHEN** o payload contém `message.ephemeralMessage` sem campo `message` interno
- **THEN** o sistema DEVE retornar o objeto `ephemeralMessage` como mensagem (melhor esforço)
- **THEN** se nenhum tipo reconhecível for encontrado, o evento DEVE ser descartado silenciosamente (sem erro)

#### Scenario: Mensagem sem wrapper (comportamento existente preservado)
- **WHEN** o payload contém `message.imageMessage` diretamente (sem wrapper)
- **THEN** o comportamento DEVE ser idêntico ao atual — sem regressão

---

### Requirement: Suporte a ptvMessage (vídeo nota redondo)

O sistema DEVE tratar `ptvMessage` como vídeo no fluxo inbound Cloud mode.

#### Scenario: Vídeo nota (ptvMessage) recebido
- **WHEN** o whatsmeow entrega payload com `message.ptvMessage` contendo `directPath`, `mediaKey`, etc.
- **THEN** o sistema DEVE baixar a mídia, fazer upload no MinIO e entregar ao Chatwoot como `type: "video"`
- **THEN** o campo `video` DEVE conter `{ "link": "<url>", "mime_type": "<mime>" }`

#### Scenario: ptvMessage sem directPath
- **WHEN** o payload `ptvMessage` não possui `directPath` preenchido
- **THEN** o sistema DEVE descartar a mensagem de mídia e logar warning
- **THEN** NÃO DEVE enviar webhook com link inválido ao Chatwoot

---

### Requirement: Suporte a contactsArrayMessage (múltiplos contatos)

O sistema DEVE converter `contactsArrayMessage` (múltiplos contatos) para o formato Cloud API `contacts` array.

#### Scenario: Múltiplos contatos em uma mensagem
- **WHEN** o whatsmeow entrega payload com `message.contactsArrayMessage.contacts` contendo 2 ou mais vCards
- **THEN** o sistema DEVE construir um payload Cloud API com `type: "contacts"` e array `contacts` com todos os vCards parseados
- **THEN** o webhook DEVE ser entregue ao Chatwoot com todos os contatos na lista `contacts`

#### Scenario: contactsArrayMessage com um único contato
- **WHEN** o array `contacts` tem exatamente 1 elemento
- **THEN** o comportamento DEVE ser equivalente a um `contactMessage` singular
- **THEN** o webhook DEVE ter `type: "contacts"` com array de 1 elemento

#### Scenario: contactsArrayMessage com vCard inválido
- **WHEN** um dos vCards no array está vazio ou malformado
- **THEN** o sistema DEVE incluir os contatos válidos e ignorar os inválidos (best-effort)
- **THEN** se nenhum contato válido sobrar, o sistema DEVE descartar o evento

---

### Requirement: Conversão de respostas interativas do usuário para texto

O sistema DEVE converter respostas a mensagens interativas (`listResponseMessage`, `buttonsResponseMessage`, `templateButtonReplyMessage`) para mensagens de texto legíveis e entregá-las ao Chatwoot como `type: "text"`.

#### Scenario: Usuário seleciona item de lista (listResponseMessage)
- **WHEN** o whatsmeow entrega `message.listResponseMessage` com `title = "Suporte Técnico"` e `singleSelectReply.selectedRowId = "row_1"`
- **THEN** o sistema DEVE formatar como texto `[Lista] Suporte Técnico`
- **THEN** o webhook Cloud API DEVE ter `type: "text"` com `text.body: "[Lista] Suporte Técnico"`

#### Scenario: Usuário clica em botão (buttonsResponseMessage)
- **WHEN** o whatsmeow entrega `message.buttonsResponseMessage` com `selectedDisplayText = "Confirmar"` ou `selectedButtonId = "btn_confirm"`
- **THEN** o sistema DEVE formatar como texto `[Botão] Confirmar`
- **THEN** o webhook Cloud API DEVE ter `type: "text"` com `text.body: "[Botão] Confirmar"`

#### Scenario: Usuário clica em botão de template (templateButtonReplyMessage)
- **WHEN** o whatsmeow entrega `message.templateButtonReplyMessage` com `selectedDisplayText = "Sim"` e `selectedId = "yes"`
- **THEN** o sistema DEVE formatar como texto `[Botão] Sim`
- **THEN** o webhook Cloud API DEVE ter `type: "text"` com `text.body: "[Botão] Sim"`

#### Scenario: Resposta interativa sem texto legível
- **WHEN** `selectedDisplayText` e `selectedButtonId` estão ambos vazios
- **THEN** o sistema DEVE descartar o evento (sem enviar webhook vazio ao Chatwoot)

---

### Requirement: Extração de payload PIX de nativeFlowMessage

O sistema DEVE extrair informações de pagamento PIX de mensagens `nativeFlowMessage` (contidas dentro de `interactiveMessage`) e entregá-las ao Chatwoot como texto formatado.

#### Scenario: Mensagem PIX com chave estática (pix_static_code)
- **WHEN** o whatsmeow entrega `message.interactiveMessage.nativeFlowMessage.buttons[0].buttonParamsJson` contendo `payment_settings[0].type = "pix_static_code"` com `merchant_name = "Loja ABC"`, `key_type = "CPF"`, `key = "123.456.789-00"`
- **THEN** o sistema DEVE desembrulhar e formatar como:
  ```
  *Loja ABC*
  Chave PIX tipo *CPF*: 123.456.789-00
  ```
- **THEN** o webhook Cloud API DEVE ter `type: "text"` com esse body

#### Scenario: Mensagem PIX com chave dinâmica (pix_dynamic_code)
- **WHEN** `payment_settings[0].type = "pix_dynamic_code"`
- **THEN** o sistema DEVE extrair e formatar da mesma forma que o estático

#### Scenario: nativeFlowMessage sem payment_settings (tipo desconhecido)
- **WHEN** o `buttonParamsJson` não contém `payment_settings` ou o tipo não é PIX
- **THEN** o sistema DEVE descartar o evento silenciosamente (sem enviar webhook)

#### Scenario: nativeFlowMessage com buttonParamsJson inválido (JSON malformado)
- **WHEN** `buttonParamsJson` não é JSON válido
- **THEN** o sistema DEVE logar warning e descartar o evento

---

## MODIFIED Requirements

### Requirement: Conversão de mensagem de mídia para formato Cloud API webhook

O sistema DEVE converter mensagens de mídia (image, video, audio, document, sticker, **ptvMessage**) para formato Cloud API, incluindo URL de download da mídia.

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

#### Scenario: Vídeo nota redondo (ptvMessage)
- **WHEN** uma mensagem `ptvMessage` chega (vídeo circular)
- **THEN** o sistema DEVE processar identicamente a um `videoMessage`
- **THEN** o campo `video` DEVE conter `{ "link": "<url>", "mime_type": "<mime>" }`

#### Scenario: Falha no download de mídia
- **WHEN** o download da mídia falha
- **THEN** o sistema DEVE logar warning com `component: "chatwoot"` e NÃO enviar o webhook
- **THEN** a falha NÃO DEVE impedir o processamento de outras mensagens

#### Scenario: MinIO indisponível
- **WHEN** o upload para MinIO falha
- **THEN** o sistema DEVE logar warning e tentar enviar o webhook apenas com o texto/caption (sem mídia)
