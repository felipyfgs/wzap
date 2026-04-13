## ADDED Requirements

### Requirement: Log de Message received com tipo de mensagem
O log INFO de "Message received" em `wa/events.go` DEVE incluir os campos `msgType` (tipo da mensagem: text, image, video, audio, document, sticker, contact, location, list, buttons, poll, reaction, unknown) e `mediaType` (mimetype, quando aplicável).

#### Scenario: Mensagem de texto recebida
- **WHEN** uma mensagem de texto é recebida
- **THEN** o log DEVE conter `msgType=text` e NÃO DEVE conter `mediaType`

#### Scenario: Mensagem de imagem recebida
- **WHEN** uma mensagem de imagem é recebida
- **THEN** o log DEVE conter `msgType=image` e `mediaType=image/jpeg` (ou mimetype correspondente)

#### Scenario: Mensagem de áudio recebida
- **WHEN** uma mensagem de áudio é recebida
- **THEN** o log DEVE conter `msgType=audio` e `mediaType=audio/ogg; codecs=opus` (ou mimetype correspondente)

### Requirement: Log de MediaRetry com message ID
O log de `MediaRetry` DEVE incluir o campo `mid` com o message ID do retry.

#### Scenario: Evento MediaRetry recebido
- **WHEN** um evento MediaRetry é recebido
- **THEN** o log DEBUG DEVE conter o campo `mid` com o ID da mensagem

### Requirement: Logs de eventos de chamada com dados do chamador
Os eventos de chamada (`CallOffer`, `CallAccept`, `CallTerminate`, `CallOfferNotice`, `CallPreAccept`, `CallReject`, `CallTransport`) DEVEM logar o campo `from` (JID do chamador) quando disponível e `callID` quando disponível no evento.

#### Scenario: CallOffer recebido
- **WHEN** um evento CallOffer é recebido
- **THEN** o log DEVE conter `from` (JID do chamador) e `callID`

#### Scenario: CallTerminate recebido
- **WHEN** um evento CallTerminate é recebido
- **THEN** o log DEVE conter `callID` quando disponível

### Requirement: Logs de eventos de newsletter com JID
Os eventos de newsletter (`NewsletterJoin`, `NewsletterLeave`, `NewsletterMuteChange`, `NewsletterLiveUpdate`) DEVEM logar o campo `jid` com o JID do newsletter.

#### Scenario: NewsletterJoin recebido
- **WHEN** um evento NewsletterJoin é recebido
- **THEN** o log DEVE conter `jid` com o JID do newsletter

#### Scenario: NewsletterLeave recebido
- **WHEN** um evento NewsletterLeave é recebido
- **THEN** o log DEVE conter `jid` com o JID do newsletter

### Requirement: Logs de Mute/Pin/Archive com valor do estado
Os eventos `Mute`, `Pin` e `Archive` DEVEM logar dados adicionais do estado quando disponíveis (ex: `action` indicando se foi ativado ou desativado).

#### Scenario: Evento Mute recebido
- **WHEN** um evento Mute é recebido
- **THEN** o log DEVE conter `jid` e `action` (muted/unmuted baseado no campo `Action` do evento)

#### Scenario: Evento Pin recebido
- **WHEN** um evento Pin é recebido
- **THEN** o log DEVE conter `jid` e `action` (pinned/unpinned baseado no campo `Action` do evento)

#### Scenario: Evento Archive recebido
- **WHEN** um evento Archive é recebido
- **THEN** o log DEVE conter `jid` e `action` (archived/unarchived baseado no campo `Action` do evento)

### Requirement: Log de Receipt com message IDs
O log de `Receipt` recebido DEVE incluir os message IDs do receipt para permitir correlação com mensagens enviadas.

#### Scenario: Receipt de leitura recebido
- **WHEN** um evento Receipt do tipo "read" é recebido
- **THEN** o log DEVE conter `type`, `chat` (JID do chat), e `count` (quantidade de message IDs no receipt)
