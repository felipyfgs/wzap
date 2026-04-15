## Why

Mensagens de status (WhatsApp Stories) estao armazenadas na mesma tabela `wz_messages` e retornadas nos mesmos endpoints de listagem de mensagens. Isso polui o historico de conversas, dificulta a filtragem de mensagens reais e impede um gerenciamento adequado de status recebidos e enviados. O setting `IgnoreStatus` existe no modelo mas nao e aplicado em nenhum lugar do codigo.

## What Changes

- Nova tabela `wz_statuses` dedicada ao armazenamento de status (stories) separada de mensagens
- Novos endpoints REST: `POST /sessions/:id/status/{text,image,video}` (enviar), `GET /sessions/:id/status` (listar recebidos), `GET /sessions/:id/status/:senderJid` (status de contato especifico)
- Roteamento de eventos no `internal/wa/events.go`: mensagens com `Chat.Server == BroadcastServer` devem ser despachadas via callback dedicado `OnStatusReceived` em vez de `OnMessageReceived`
- Implementacao do setting `IgnoreStatus` no filtro de eventos
- **BREAKING**: `GET /sessions/:id/messages` nao retornara mais mensagens de status
- Nova pagina dedicada de status no frontend (Nuxt) com visual de stories
- Modal dedicado para envio de status, separado do modal de mensagens
- Migracao de dados existentes de `wz_messages` (chat_jid LIKE 'status@%') para `wz_statuses`

## Capabilities

### New Capabilities
- `status-storage`: armazenamento dedicado de WhatsApp Stories (tabela `wz_statuses`, repo, persistencia de status enviados e recebidos)
- `status-api`: endpoints REST para envio e listagem de status, roteamento de eventos whatsmeow
- `status-frontend`: pagina e componentes frontend para visualizacao e envio de status

### Modified Capabilities
(nenhuma capability existente com mudanca de requisitos de spec)

## Impact

- **Backend**: novos arquivos em `model/`, `repo/`, `service/`, `handler/`, `dto/`; modificacoes em `wa/events.go`, `wa/manager.go`, `service/message_status.go`, `service/history.go`, `repo/message.go`, `server/router.go`
- **Frontend**: nova pagina `pages/sessions/[id]/status.vue`, novos componentes, novo composable `useStatus.ts`, modificacoes no layout e no `SendMessageModal`
- **Database**: nova tabela `wz_statuses` (migration 008), migracao de dados existentes
- **API**: novos endpoints de status; **BREAKING** — `GET /messages` exclui status
- **Integracoes**: Chatwoot ja ignora `status@broadcast`, mas precisa verificar `status@` generico
