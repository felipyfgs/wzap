## ADDED Requirements

### Requirement: Filtrar senderKeyDistributionMessage no handler live

O handler de eventos live (`handleEvent`) MUST ignorar mensagens `*events.Message` que contêm apenas `senderKeyDistributionMessage` sem conteúdo real de mensagem.

A filtragem MUST verificar que `extractMessageContent(v.Message)` retorna `"unknown"` E o proto contém `GetSenderKeyDistributionMessage() != nil`, para evitar filtrar mensagens que tenham conteúdo real combinado.

#### Scenario: senderKeyDistributionMessage standalone é ignorado

- **WHEN** uma mensagem `*events.Message` chega com apenas `senderKeyDistributionMessage` (sem texto, mídia ou outro conteúdo)
- **THEN** o handler faz `return` sem persistir, sem despachar webhook, sem logar como "Message received"

#### Scenario: Mensagem com senderKeyDistribution e conteúdo real é processada

- **WHEN** uma mensagem `*events.Message` chega com `senderKeyDistributionMessage` E um `imageMessage` ou outro tipo de conteúdo real
- **THEN** o handler processa normalmente como `image` (ou o tipo correspondente)
- **THEN** a mensagem é persistida e despachada via webhook

#### Scenario: Log de filtragem em DEBUG

- **WHEN** uma `senderKeyDistributionMessage` standalone é filtrada
- **THEN** o sistema loga em nível DEBUG: `"Sender key distribution message filtered"` com campos `session` e `mid`

### Requirement: Não persistir senderKeyDistributionMessage no history sync

A função `buildHistoryMessage` em `internal/service/history.go` MUST filtrar mensagens que contêm apenas `senderKeyDistributionMessage` sem conteúdo real, retornando `nil`.

#### Scenario: senderKeyDistribution do histórico é filtrado

- **WHEN** uma mensagem do history sync tem proto com apenas `senderKeyDistributionMessage`
- **THEN** `buildHistoryMessage` retorna `nil`
- **THEN** a mensagem não é salva em `wz_messages`
