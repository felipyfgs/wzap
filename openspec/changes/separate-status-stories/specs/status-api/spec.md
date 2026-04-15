## ADDED Requirements

### Requirement: Endpoint POST /sessions/:sessionId/status/text
O sistema SHALL expor endpoint `POST /sessions/:sessionId/status/text` para enviar texto como WhatsApp Story. O request body SHALL conter `text` (string, obrigatorio). O response SHALL ser `{ data: { mid: string } }` com codigo HTTP 200.

#### Scenario: Envio bem-sucedido
- **WHEN** um POST e enviado com `{ "text": "Hello world" }` para uma sessao conectada
- **THEN** o status e enviado via whatsmeow, persistido em `wz_statuses` e retorna HTTP 200 com o ID da mensagem

#### Scenario: Sessao desconectada
- **WHEN** um POST e enviado para uma sessao desconectada
- **THEN** retorna HTTP 503 com mensagem de erro generica

### Requirement: Endpoint POST /sessions/:sessionId/status/image
O sistema SHALL expor endpoint `POST /sessions/:sessionId/status/image` para enviar imagem como WhatsApp Story. O request body SHALL conter `base64` (string, obrigatorio se `url` ausente), `url` (string, obrigatorio se `base64` ausente), `caption` (string, opcional), `mimeType` (string, obrigatorio). O response SHALL ser `{ data: { mid: string } }` com codigo HTTP 200.

#### Scenario: Envio de imagem por base64
- **WHEN** um POST e enviado com `{ "base64": "...", "mimeType": "image/jpeg" }`
- **THEN** a imagem e enviada como status, persistida em `wz_statuses` com `status_type=status_image`, e retorna HTTP 200

### Requirement: Endpoint POST /sessions/:sessionId/status/video
O sistema SHALL expor endpoint `POST /sessions/:sessionId/status/video` para enviar video como WhatsApp Story. O contrato e identico ao endpoint de imagem, porem com `mimeType` de video.

#### Scenario: Envio de video
- **WHEN** um POST e enviado com `{ "url": "https://example.com/video.mp4", "mimeType": "video/mp4" }`
- **THEN** o video e enviado como status, persistido em `wz_statuses` com `status_type=status_video`, e retorna HTTP 200

### Requirement: Endpoint GET /sessions/:sessionId/status
O sistema SHALL expor endpoint `GET /sessions/:sessionId/status` para listar status recebidos. Suporta query params: `limit` (int, default 50, max 200), `offset` (int, default 0). O response SHALL retornar array de `model.Status` ordenado por timestamp DESC.

#### Scenario: Listar status
- **WHEN** GET e chamado sem query params
- **THEN** retorna ate 50 status ordenados por timestamp DESC

#### Scenario: Paginacao
- **WHEN** GET e chamado com `?limit=10&offset=20`
- **THEN** retorna ate 10 status a partir do offset 20

### Requirement: Endpoint GET /sessions/:sessionId/status/:senderJid
O sistema SHALL expor endpoint `GET /sessions/:sessionId/status/:senderJid` para listar todos os status de um contato especifico. O response SHALL retornar array de `model.Status` ordenado por timestamp ASC (ordem cronologica de stories).

#### Scenario: Status de um contato
- **WHEN** GET e chamado com `/status/5511999999999@s.whatsapp.net`
- **THEN** retorna todos os status desse contato em ordem cronologica

### Requirement: Roteamento de eventos whatsmeow para status
O `wa.Manager` SHALL verificar se uma mensagem recebida tem `Chat.Server == types.BroadcastServer` e, se sim, invocar o callback `OnStatusReceived` em vez de `OnMessageReceived`. O callback SHALL ter a assinatura: `StatusReceivedFunc(sessionID, msgID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw any)`.

#### Scenario: Status recebido roteado
- **WHEN** whatsmeow despacha um `events.Message` com `Chat.Server == BroadcastServer`
- **THEN** `OnStatusReceived` e invocado e `OnMessageReceived` nao e invocado

#### Scenario: Mensagem normal nao afetada
- **WHEN** whatsmeow despacha um `events.Message` com `Chat.Server != BroadcastServer`
- **THEN** `OnMessageReceived` e invocado normalmente

### Requirement: Implementacao do setting IgnoreStatus
O sistema SHALL respeitar o setting `SessionSettings.IgnoreStatus`. Quando `IgnoreStatus` e `true`, mensagens de status (Chat.Server == BroadcastServer) SHALL ser descartadas silenciosamente (log debug, sem persistir, sem despachar evento).

#### Scenario: IgnoreStatus ativado
- **WHEN** uma sessao tem `IgnoreStatus: true` e um status e recebido
- **THEN** o status e descartado sem persistencia e sem dispatch de evento

#### Scenario: IgnoreStatus desativado
- **WHEN** uma sessao tem `IgnoreStatus: false` e um status e recebido
- **THEN** o status e persistido e o evento e despachado normalmente

### Requirement: Historico de status
O `HistoryService` SHALL ignorar conversas com JID iniciando em `status@` durante persistencia de history sync em `wz_messages`, evitando poluir a tabela de mensagens.

#### Scenario: History sync com conversa de status
- **WHEN** um history sync contem uma conversa com `chat_jid` iniciando em `status@`
- **THEN** as mensagens dessa conversa nao sao persistidas em `wz_messages`
