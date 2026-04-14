## ADDED Requirements

### Requirement: Migração SQL para limpar mensagens unknown inválidas

O sistema MUST incluir uma migração SQL (`007_cleanup_unknown_messages.up.sql`) que remove registros de `wz_messages` classificados como `msg_type = 'unknown'` que são comprovadamente lixo de protocolo.

A migração MUST deletar registros que atendam **qualquer** das condições:
1. `msg_type = 'unknown'` E `raw::text LIKE '%protocolMessage%'`
2. `msg_type = 'unknown'` E `raw::text LIKE '%messageStubType%'`
3. `msg_type = 'unknown'` E `raw::text LIKE '%senderKeyDistributionMessage%'` E `raw::text NOT LIKE '%conversation%'` E `raw::text NOT LIKE '%imageMessage%'` E `raw::text NOT LIKE '%videoMessage%'` E `raw::text NOT LIKE '%audioMessage%'` E `raw::text NOT LIKE '%documentMessage%'`

#### Scenario: Registros protocolMessage são removidos

- **WHEN** a migração roda em um banco com 749 registros `unknown` contendo `protocolMessage`
- **THEN** todos os 749 registros são removidos
- **THEN** mensagens com `msg_type != 'unknown'` não são afetadas

#### Scenario: Registros messageStubType são removidos

- **WHEN** a migração roda em um banco com 3.722 registros `unknown` contendo `messageStubType`
- **THEN** todos os 3.722 registros são removidos

#### Scenario: senderKeyDistribution sem conteúdo real é removido

- **WHEN** a migração roda em um banco com registros `unknown` contendo apenas `senderKeyDistributionMessage`
- **THEN** os registros são removidos
- **THEN** registros que contenham `senderKeyDistributionMessage` junto com `imageMessage` ou outro conteúdo real não são removidos

#### Scenario: Mensagens unknown legítimas são preservadas

- **WHEN** a migração roda em um banco com registros `unknown` que não contêm `protocolMessage`, `messageStubType` ou `senderKeyDistributionMessage`
- **THEN** esses registros são preservados

### Requirement: Migração down é no-op

A migração down (`007_cleanup_unknown_messages.down.sql`) MUST ser um no-op (sem operação), pois os dados removidos são lixo de protocolo e não precisam ser restaurados.

#### Scenario: Rollback da migração

- **WHEN** a migração down é executada
- **THEN** nenhuma alteração é feita no banco
- **THEN** a migração completa sem erro
