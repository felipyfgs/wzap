## Why

O wzap já persiste mensagens avulsas em `wz_messages`, mas ainda não possui uma camada canônica para armazenar conversas e histórico completo recebidos do `whatsmeow`. Isso impede reaproveitar com segurança o `HistorySync` real do protocolo e dificulta uma importação futura e confiável para o Chatwoot.

## What Changes

- Criar uma camada canônica de persistência para conversas do WhatsApp no domínio do WZAP, separada das tabelas internas `whatsmeow_*`
- Persistir metadados de chat recebidos via `events.HistorySync` e também consolidar estado vindo do fluxo ao vivo
- Enriquecer a persistência de mensagens para diferenciar origem (`live` e `history_sync`) e registrar metadados necessários para reprocessamento futuro
- Adicionar um ponto explícito no `wa.Manager` para ingestão do `HistorySync` completo, sem depender do payload resumido hoje enviado para webhook/NATS
- Preparar a base de dados e os repositórios para que o importador do Chatwoot possa futuramente ler histórico ordenado e idempotente a partir do banco do WZAP

## Capabilities

### New Capabilities
- `canonical-chat-store`: persistência canônica de chats/conversas do WhatsApp no domínio do WZAP, incluindo estado e metadados relevantes por sessão
- `whatsmeow-history-sync-persistence`: ingestão e persistência completa de `events.HistorySync` do `whatsmeow`, com armazenamento idempotente de mensagens e chats históricos

### Modified Capabilities

## Impact

- `internal/wa/events.go` e `internal/wa/manager.go` — novo ponto de ingestão do `HistorySync` completo
- `internal/service/history.go` — evolução de persistência para chats e histórico, além de mensagens do fluxo live
- `internal/repo/message.go` e novos repositórios/modelos de chat — leitura e escrita canônicas no banco
- `internal/model/` — novos tipos de domínio para chat e metadados de ingestão
- `migrations/` — nova tabela `wz_chats` e ajustes em `wz_messages`
- `internal/integrations/chatwoot/service.go` — etapa futura poderá consumir essa base canônica em vez do placeholder atual de import

## Não-objetivos

- Não implementar, nesta change, o importador completo do Chatwoot com replay de mensagens
- Não usar tabelas `whatsmeow_*` como fonte principal de produto
- Não alterar o contrato atual de webhook/NATS para enviar o blob completo de `HistorySync`

## Riscos e mitigações

- Duplicidade entre mensagens do fluxo ao vivo e do `HistorySync` → usar upsert idempotente por identificador da mensagem e sessão
- Diferenças entre `LID` e `PN` nos JIDs → armazenar aliases e aproveitar os mapeamentos já expostos pelo `whatsmeow`
- Chunks de `HistorySync` fora de ordem → registrar `chunkOrder`, origem e timestamps para reprocessamento consistente
- Escopo crescer para “import completo” cedo demais → limitar esta change à fundação canônica de persistência
