## ADDED Requirements

### Requirement: Toggle de Announce-only em grupo
O dashboard SHALL permitir ativar/desativar o modo announce-only de um grupo via `POST /sessions/:sessionId/groups/announce`.

#### Scenario: Ativar announce-only
- **WHEN** o usuário ativa o switch "Announce Only" na aba Settings do grupo
- **THEN** faz `POST /groups/announce` com `{ groupJid, announce: true }` e exibe toast de sucesso

#### Scenario: Desativar announce-only
- **WHEN** o usuário desativa o switch "Announce Only"
- **THEN** faz `POST /groups/announce` com `{ groupJid, announce: false }` e exibe toast de sucesso

#### Scenario: Erro na alteração
- **WHEN** a requisição falha
- **THEN** o switch DEVE reverter para o estado anterior e exibir toast de erro

### Requirement: Toggle de grupo trancado (locked)
O dashboard SHALL permitir ativar/desativar o modo locked de um grupo via `POST /sessions/:sessionId/groups/locked`.

#### Scenario: Ativar locked
- **WHEN** o usuário ativa o switch "Locked" na aba Settings do grupo
- **THEN** faz `POST /groups/locked` com `{ groupJid, locked: true }` e exibe toast de sucesso

#### Scenario: Desativar locked
- **WHEN** o usuário desativa o switch "Locked"
- **THEN** faz `POST /groups/locked` com `{ groupJid, locked: false }` e exibe toast de sucesso

### Requirement: Toggle de aprovação para entrada (join-approval)
O dashboard SHALL permitir ativar/desativar a aprovação para entrada em um grupo via `POST /sessions/:sessionId/groups/join-approval`.

#### Scenario: Ativar join-approval
- **WHEN** o usuário ativa o switch "Join Approval" na aba Settings do grupo
- **THEN** faz `POST /groups/join-approval` com `{ groupJid, enabled: true }` e exibe toast de sucesso

#### Scenario: Desativar join-approval
- **WHEN** o usuário desativa o switch "Join Approval"
- **THEN** faz `POST /groups/join-approval` with `{ groupJid, enabled: false }` e exibe toast de sucesso

### Requirement: Posicionamento dos toggles
Os três novos toggles DEVEM ser exibidos na aba Settings do `GroupSlideover`, junto com o toggle de ephemeral já existente.

#### Scenario: Layout dos toggles
- **WHEN** o usuário abre a aba Settings de um grupo
- **THEN** os toggles DEVEM aparecer na ordem: Ephemeral, Announce Only, Locked, Join Approval
- **THEN** cada toggle DEVE ter label descritivo e description explicando o efeito
