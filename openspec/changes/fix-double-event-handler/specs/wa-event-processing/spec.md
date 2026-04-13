## ADDED Requirements

### Requirement: Processamento único de eventos WhatsApp por sessão
O sistema SHALL processar cada evento WhatsApp recebido exatamente uma vez por sessão. Nenhum handler deve ser registrado mais de uma vez para o mesmo cliente whatsmeow durante o ciclo de vida de uma sessão.

#### Scenario: Mensagem recebida não gera log duplicado
- **WHEN** uma sessão recebe uma mensagem WhatsApp
- **THEN** o log "Message received" aparece exatamente uma vez com o `mid` da mensagem

#### Scenario: Webhook não é disparado duas vezes para o mesmo evento
- **WHEN** uma sessão recebe um evento de tipo `Message`
- **THEN** o dispatcher de webhooks é chamado exatamente uma vez para aquele evento

#### Scenario: handleMessage do Chatwoot não é chamado duas vezes para o mesmo `id`
- **WHEN** uma mensagem é recebida em uma sessão com Chatwoot configurado
- **THEN** o log `handleMessage` com o `id` da mensagem aparece exatamente uma vez

### Requirement: Extração de tipo de mensagem cobre todos os tipos conhecidos
O sistema SHALL retornar um `msgType` específico e não-genérico para todos os tipos de mensagem suportados pelo WhatsApp. Para tipos não mapeados, SHALL retornar `"unsupported"` em vez de `"unknown"`.

#### Scenario: Mensagem de texto retorna msgType correto
- **WHEN** uma mensagem do tipo texto (Conversation ou ExtendedText) é recebida
- **THEN** o campo `msgType` no log é `"text"`

#### Scenario: Mensagem de mídia retorna msgType e mediaType corretos
- **WHEN** uma mensagem de imagem, vídeo, áudio, documento ou sticker é recebida
- **THEN** o campo `msgType` é o tipo correto (ex: `"image"`, `"video"`) e `mediaType` contém o MIME type

#### Scenario: Tipo de mensagem não mapeado retorna unsupported
- **WHEN** uma mensagem de tipo ainda não mapeado em `extractMessageContent` é recebida
- **THEN** o campo `msgType` no log é `"unsupported"` (não `"unknown"`)

### Requirement: Logs de handler Chatwoot incluem campo session
O sistema SHALL incluir o campo `session` em todos os logs emitidos pelo handler de mensagens inbound do Chatwoot.

#### Scenario: Log handleMessage inclui session
- **WHEN** o handler Chatwoot processa uma mensagem inbound
- **THEN** o log `handleMessage` contém os campos `component=chatwoot`, `session`, `chat` e `id`

### Requirement: Pool assíncrono usa campo component nos logs
O sistema SHALL usar `component=async` em todos os logs emitidos por `internal/async/pool.go`.

#### Scenario: Log de pool iniciado inclui component
- **WHEN** um pool assíncrono é iniciado
- **THEN** o log contém `component=async` e o campo `pool` com o nome do pool
