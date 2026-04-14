## Why

O wzap já captura eventos de `HistorySync` do WhatsApp e persiste mensagens/chats no banco local (`wz_messages`, `wz_chats`), mas **não faz download das mídias do histórico** e **não sincroniza esse histórico com o Chatwoot**. O `importHistory` atual é um placeholder. Contatos da agenda do WhatsApp também não são priorizados — pushNames sobreescrevem nomes da agenda indiscriminadamente. Isso resulta em agentes do Chatwoot sem contexto histórico das conversas e contatos com nomes inconsistentes.

A Evolution API resolve parcialmente inserindo direto no Postgres do Chatwoot (frágil, sem mídias, sem histórico local). Nosso approach usa a API REST do Chatwoot, persiste mídias no MinIO e mantém um histórico local completo e imaculável.

## What Changes

- **Download de mídias do HistorySync**: ao receber mensagens de histórico com media (image, video, audio, document, sticker), fazer download via whatsmeow e upload para MinIO
- **Implementar `importHistory` real**: iterar mensagens do `wz_messages` (source=`history_sync`, não importadas), criar contatos/conversas/mensagens no Chatwoot via API REST, com upload de mídias do MinIO
- **Sincronização de contatos**: priorizar nome da agenda WA (FullName) sobre pushName; `handlePushName` não sobrescreve nomes já definidos; `upsertConversation` consulta `Store.Contacts` para usar nome da agenda
- **Endpoint de importação manual**: implementar `POST /sessions/{id}/integrations/chatwoot/import` (hoje retorna 501)
- **Query de mensagens não importadas**: novo método no repo para buscar mensagens por período com `imported_to_chatwoot_at IS NULL`

## Não-objetivos

- Acesso direto ao Postgres do Chatwoot (sempre via API REST)
- Importação de mensagens de grupos (apenas chats 1:1, como a Evolution)
- Importação de status/stories (`status@broadcast`)
- Re-download de mídias já armazenadas no MinIO

## Capabilities

### New Capabilities
- `history-media-download`: Download e persistência de mídias recebidas via HistorySync no MinIO
- `history-chatwoot-import`: Sincronização de mensagens históricas do DB local para o Chatwoot via API REST
- `contact-name-sync`: Hierarquia de nomes de contatos (Agenda WA > PushName > Telefone) na integração Chatwoot

### Modified Capabilities

## Riscos e Mitigações

| Risco | Mitigação |
|---|---|
| Media keys do histórico expiram (~2 semanas) | Download imediato no `PersistHistorySync`; fallback para placeholder texto |
| Volume alto de mensagens sobrecarrega API do Chatwoot | Rate limiter existente (10 msgs/s via `rateTicker`); processamento em chunks |
| `importHistory` concorrente para mesma sessão | Usar `singleflight` ou flag atômico para evitar imports simultâneos |
| Falha parcial no import deixa mensagens sem `imported_to_chatwoot_at` | Idempotente — re-executar retoma de onde parou |

## Impact

- **Código**: `internal/service/history.go`, `internal/wa/manager.go`, `internal/integrations/chatwoot/service.go`, `conversation.go`, `inbound_events.go`, `handler.go`, `internal/repo/message.go`, `internal/server/router.go`
- **APIs**: Novo endpoint funcional `POST /sessions/{id}/integrations/chatwoot/import`
- **Dependências**: Nenhuma nova — usa whatsmeow (download), MinIO client (upload), Chatwoot HTTP client (já existente)
- **DB**: Nenhuma migration — campos `source`, `imported_to_chatwoot_at`, `media_url` já existem em `wz_messages`
