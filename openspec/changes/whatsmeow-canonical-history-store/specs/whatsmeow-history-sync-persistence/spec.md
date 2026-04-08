## ADDED Requirements

### Requirement: Persistência completa do HistorySync do whatsmeow
O sistema SHALL ingerir o conteúdo completo de `events.HistorySync` do `whatsmeow` e persistir suas conversas e mensagens no banco do WZAP, sem depender do payload resumido enviado por webhook ou NATS.

#### Scenario: Blob histórico recebido pelo protocolo é ingerido localmente
- **WHEN** o `wa.Manager` receber um `events.HistorySync`
- **THEN** o sistema chama uma rotina explícita de ingestão com acesso ao objeto `waHistorySync.HistorySync`
- **THEN** as conversas e mensagens históricas do blob podem ser persistidas localmente

#### Scenario: Payload resumido continua sendo usado para webhook/NATS
- **WHEN** um `events.HistorySync` for recebido
- **THEN** o sistema continua publicando apenas o resumo do evento para webhook, WebSocket e NATS
- **THEN** o blob completo não é enviado nesses canais

### Requirement: Mensagens históricas e live compartilham armazenamento canônico idempotente
O sistema SHALL persistir mensagens do fluxo live e do `HistorySync` na base canônica de mensagens do WZAP com comportamento idempotente e metadados de origem.

#### Scenario: Mensagem histórica inédita é persistida
- **WHEN** uma mensagem presente em `HistorySync.Conversation.Messages` ainda não existir para a sessão
- **THEN** o sistema a persiste em `wz_messages`
- **THEN** registra a origem como histórica e os metadados de sincronização disponíveis

#### Scenario: Mensagem já existente é reencontrada no histórico ou no fluxo live
- **WHEN** uma mensagem com mesmo identificador e sessão já existir em `wz_messages`
- **THEN** o sistema faz upsert idempotente em vez de criar duplicidade
- **THEN** preserva o melhor conjunto de dados disponível para essa mensagem

### Requirement: Ordem e rastreabilidade da ingestão histórica
O sistema SHALL registrar metadados suficientes para reprocessar ou importar o histórico posteriormente de forma consistente.

#### Scenario: HistorySync chega em múltiplos chunks
- **WHEN** o histórico for recebido em mais de um chunk
- **THEN** o sistema persiste metadados como tipo de sync, `chunkOrder` e timestamps relevantes
- **THEN** o histórico pode ser reordenado e consumido posteriormente de forma determinística
