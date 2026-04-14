## ADDED Requirements

### Requirement: Hierarquia de nomes de contato
O sistema SHALL priorizar nomes de contato seguindo a hierarquia: Nome da Agenda WA (FullName/FirstName) > PushName > Número de telefone. A interface `ContactNameGetter` SHALL consultar os contatos armazenados no whatsmeow `Store.Contacts` para obter o nome da agenda.

#### Scenario: Contato com nome na agenda WA
- **WHEN** um contato tem `FullName` ou `FirstName` definido na agenda do WhatsApp (`Store.Contacts`)
- **THEN** o sistema usa esse nome como nome do contato no Chatwoot, ignorando `pushName`

#### Scenario: Contato sem nome na agenda, com pushName
- **WHEN** um contato não possui nome na agenda do WhatsApp, mas possui `pushName`
- **THEN** o sistema usa `pushName` como nome do contato no Chatwoot

#### Scenario: Contato sem nome na agenda e sem pushName
- **WHEN** um contato não possui nome na agenda nem pushName
- **THEN** o sistema usa o número de telefone como nome do contato no Chatwoot

### Requirement: handlePushName não sobrescreve nome da agenda
O handler de evento `PushName` SHALL verificar se o contato já possui um nome não-numérico definido no Chatwoot antes de atualizar. Se o nome existente não for um número de telefone, o pushName SHALL ser ignorado.

#### Scenario: PushName recebido para contato sem nome definido
- **WHEN** evento `PushName` chega e o contato no Chatwoot tem nome igual ao número de telefone ou vazio
- **THEN** o sistema atualiza o nome do contato para o pushName

#### Scenario: PushName recebido para contato já com nome da agenda
- **WHEN** evento `PushName` chega e o contato no Chatwoot já tem um nome diferente do número de telefone
- **THEN** o sistema ignora a atualização e mantém o nome existente

### Requirement: upsertConversation usa nome da agenda
Ao criar ou atualizar uma conversa no Chatwoot, o sistema SHALL consultar `ContactNameGetter.GetContactName` antes de usar `pushName` como nome do contato.

#### Scenario: Criação de conversa com nome da agenda disponível
- **WHEN** `upsertConversation` é chamado para um chatJID cujo contato tem nome na agenda WA
- **THEN** o nome do contato criado no Chatwoot usa o nome da agenda WA em vez do pushName

#### Scenario: Criação de conversa sem nome na agenda
- **WHEN** `upsertConversation` é chamado para um chatJID cujo contato não tem nome na agenda WA
- **THEN** o nome do contato usa pushName se disponível, ou o número de telefone como fallback
