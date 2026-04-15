## ADDED Requirements

### Requirement: Tabela dedicada wz_statuses
O sistema SHALL criar uma tabela `wz_statuses` no PostgreSQL para armazenar WhatsApp Stories separadamente das mensagens. A tabela SHALL conter: `id` (VARCHAR 100), `session_id` (VARCHAR 100, FK para `wz_sessions`), `sender_jid` (VARCHAR 255), `from_me` (BOOLEAN), `status_type` (VARCHAR 50), `body` (TEXT), `media_type` (VARCHAR 50), `media_url` (TEXT), `raw` (JSONB), `timestamp` (TIMESTAMPTZ), `created_at` (TIMESTAMPTZ). A chave primaria SHALL ser composta por `(id, session_id)`.

#### Scenario: Criacao da tabela
- **WHEN** a migration `008_statuses.up.sql` e executada
- **THEN** a tabela `wz_statuses` existe com todos os campos, constraints e indices

### Requirement: Indices de performance
O sistema SHALL criar indices em `wz_statuses` para suportar os padroes de acesso: `(session_id, timestamp DESC)` para listagem por tempo e `(session_id, sender_jid, timestamp DESC)` para listagem por contato.

#### Scenario: Queries otimizadas
- **WHEN** status sao listados por sessao ou por contato
- **THEN** as queries utilizam os indices criados

### Requirement: Model Status
O sistema SHALL definir um struct `model.Status` em `internal/model/status.go` mapeando os campos da tabela `wz_statuses`, com tags JSON para serializacao.

#### Scenario: Struct mapeado
- **WHEN** um `model.Status` e instanciado
- **THEN** os campos correspondem as colunas da tabela `wz_statuses`

### Requirement: Repositorio StatusRepo
O sistema SHALL definir uma interface `StatusRepo` em `internal/repo/status.go` com os metodos: `Save(ctx, *model.Status) error`, `FindBySession(ctx, sessionID, limit, offset) ([]model.Status, error)`, `FindBySender(ctx, sessionID, senderJID) ([]model.Status, error)`, `DeleteBySender(ctx, sessionID, senderJID) error`. A implementacao `StatusRepository` SHALL usar pgx para acesso ao PostgreSQL.

#### Scenario: Salvar status recebido
- **WHEN** `Save` e chamado com um status valido
- **THEN** o registro e inserido na tabela `wz_statuses` (upsert on conflict)

#### Scenario: Listar status por sessao
- **WHEN** `FindBySession` e chamado com sessionID, limit 50, offset 0
- **THEN** retorna ate 50 status ordenados por timestamp DESC

#### Scenario: Listar status por contato
- **WHEN** `FindBySender` e chamado com sessionID e senderJID
- **THEN** retorna todos os status daquele contato ordenados por timestamp ASC

### Requirement: Migracao de dados existentes
A migration SHALL copiar registros de `wz_messages` onde `chat_jid LIKE 'status@%'` para `wz_statuses`, usando `sender_jid` (ou `chat_jid` como fallback) e `msg_type` como `status_type`. Apos a copia, os registros originais SHALL ser removidos de `wz_messages`.

#### Scenario: Dados migrados
- **WHEN** a migration e executada em um banco com mensagens de status
- **THEN** os registros aparecem em `wz_statuses` e nao mais em `wz_messages`

### Requirement: Filtro de status em FindBySession
O `MessageRepo.FindBySession` SHALL adicionar condicao `AND chat_jid NOT LIKE 'status@%'` para excluir status da listagem de mensagens.

#### Scenario: Mensagens sem status
- **WHEN** `FindBySession` e chamado
- **THEN** mensagens com `chat_jid` iniciando com `status@` nao sao retornadas
