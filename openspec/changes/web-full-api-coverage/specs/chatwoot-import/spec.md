## ADDED Requirements

### Requirement: Botão de importação de histórico Chatwoot
O dashboard SHALL permitir importar histórico de conversas para o Chatwoot via `POST /sessions/:sessionId/integrations/chatwoot/import`.

#### Scenario: Import com configuração ativa
- **WHEN** a integração Chatwoot está configurada (`hasConfig: true`)
- **THEN** o `ChatwootConfigCard` DEVE exibir um botão "Import History" abaixo da configuração

#### Scenario: Import com confirmação
- **WHEN** o usuário clica em "Import History"
- **THEN** o sistema DEVE exibir modal de confirmação explicando que o processo pode demorar e que importará mensagens históricas para o Chatwoot
- **THEN** ao confirmar, faz `POST /integrations/chatwoot/import` e exibe toast "Import started"

#### Scenario: Import sem configuração
- **WHEN** a integração Chatwoot NÃO está configurada
- **THEN** o botão "Import History" NÃO DEVE ser exibido

#### Scenario: Erro na importação
- **WHEN** a requisição falha
- **THEN** o sistema exibe toast de erro "Failed to start import"

#### Scenario: Feedback de importação em andamento
- **WHEN** o import é iniciado com sucesso
- **THEN** o botão DEVE ficar em estado loading por 3 segundos e depois voltar ao normal
- **THEN** um badge "Import requested" DEVE ser exibido temporariamente
