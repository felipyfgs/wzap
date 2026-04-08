## ADDED Requirements

### Requirement: Persistência canônica de chats do WhatsApp
O sistema SHALL manter uma representação canônica de chats do WhatsApp no banco do WZAP, separada das tabelas internas `whatsmeow_*`, para consolidar metadados relevantes por sessão e permitir reprocessamento futuro.

#### Scenario: Chat histórico é persistido a partir de HistorySync
- **WHEN** o `wa.Manager` receber um `events.HistorySync` contendo uma `Conversation`
- **THEN** o sistema persiste ou atualiza um registro canônico de chat associado à sessão e ao `chat_jid`
- **THEN** o registro preserva metadados de conversa necessários para uso futuro, como nome, flags e timestamps disponíveis

#### Scenario: Chat live atualiza estado canônico existente
- **WHEN** uma mensagem ao vivo ou evento relevante trouxer informações mais recentes de um chat já persistido
- **THEN** o sistema atualiza o registro canônico correspondente sem criar duplicidade
- **THEN** o chat continua identificável de forma única por sessão e chat

#### Scenario: Tabelas internas do whatsmeow não são tratadas como modelo de produto
- **WHEN** o sistema precisar consultar dados canônicos de chat para recursos do WZAP
- **THEN** ele utiliza as tabelas do domínio do WZAP em vez de depender diretamente de `whatsmeow_contacts` ou `whatsmeow_chat_settings`

### Requirement: Metadados de alias e identidade do chat
O sistema SHALL preservar metadados suficientes para reconciliar diferenças entre identificadores de chat do WhatsApp, incluindo cenários com `PN` e `LID`, sem perder a referência canônica do chat.

#### Scenario: HistorySync expõe identificadores alternativos do mesmo chat
- **WHEN** uma `Conversation` histórica trouxer campos como `pnJID`, `lidJID` ou identificadores equivalentes
- **THEN** o sistema registra esses aliases junto do chat canônico
- **THEN** a persistência futura pode reutilizar esses aliases para reconciliação e import
