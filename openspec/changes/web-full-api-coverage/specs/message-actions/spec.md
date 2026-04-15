## ADDED Requirements

### Requirement: Editar mensagem enviada
O dashboard SHALL permitir editar o conteúdo de uma mensagem previamente enviada pelo usuário via `POST /sessions/:sessionId/messages/edit`.

#### Scenario: Edição bem-sucedida
- **WHEN** o usuário clica em "Edit" no dropdown de ações de uma mensagem com `fromMe: true`
- **THEN** o sistema abre um modal com o conteúdo atual da mensagem em um textarea editável
- **THEN** ao submeter, o sistema faz `POST /messages/edit` com `{ messageId, chatJid, body }` e exibe toast de sucesso

#### Scenario: Mensagem não editável
- **WHEN** o usuário visualiza uma mensagem com `fromMe: false`
- **THEN** a opção "Edit" NÃO DEVE aparecer no dropdown de ações

### Requirement: Deletar mensagem
O dashboard SHALL permitir deletar uma mensagem via `POST /sessions/:sessionId/messages/delete`.

#### Scenario: Deleção com confirmação
- **WHEN** o usuário clica em "Delete" no dropdown de ações de uma mensagem
- **THEN** o sistema DEVE exibir um modal de confirmação antes de executar
- **THEN** ao confirmar, faz `POST /messages/delete` com `{ messageId, chatJid }` e exibe toast de sucesso

#### Scenario: Cancelar deleção
- **WHEN** o usuário clica em "Cancel" no modal de confirmação
- **THEN** nenhuma requisição é feita e o modal fecha

### Requirement: Reagir a mensagem
O dashboard SHALL permitir enviar uma reação emoji a uma mensagem via `POST /sessions/:sessionId/messages/reaction`.

#### Scenario: Enviar reação
- **WHEN** o usuário clica em "React" no dropdown de ações de uma mensagem
- **THEN** o sistema abre um modal com input de texto para emoji
- **THEN** ao submeter, faz `POST /messages/reaction` com `{ messageId, chatJid, reaction }` e exibe toast de sucesso

#### Scenario: Remover reação
- **WHEN** o usuário submete o modal de reação com campo vazio
- **THEN** o sistema envia reação vazia (remove a reação existente)

### Requirement: Marcar mensagem como lida via API de mensagens
O dashboard SHALL permitir marcar mensagens específicas como lidas via `POST /sessions/:sessionId/messages/read`.

#### Scenario: Marcar como lida
- **WHEN** o usuário clica em "Mark Read" no dropdown de ações de uma mensagem
- **THEN** faz `POST /messages/read` com `{ chatJid, messageIds: [messageId] }` e exibe toast de sucesso

### Requirement: Definir presença em chat
O dashboard SHALL permitir definir presença (typing/recording) via `POST /sessions/:sessionId/messages/presence`.

#### Scenario: Enviar presença
- **WHEN** o usuário clica em "Set Presence" no dropdown de ações de uma mensagem
- **THEN** o sistema abre um modal com seletor de tipo de presença (composing, recording, paused)
- **THEN** ao submeter, faz `POST /messages/presence` com `{ chatJid, type }` e exibe toast de sucesso

### Requirement: Encaminhar mensagem
O dashboard SHALL permitir encaminhar uma mensagem para outro contato via `POST /sessions/:sessionId/messages/forward`.

#### Scenario: Encaminhar para contato
- **WHEN** o usuário clica em "Forward" no dropdown de ações de uma mensagem
- **THEN** o sistema abre um modal com input de telefone/JID do destinatário
- **THEN** ao submeter, faz `POST /messages/forward` com `{ messageId, chatJid, to }` e exibe toast de sucesso
