## Why

O dashboard Nuxt cobre ~80% das rotas da API REST, mas deixa de fora 16 endpoints de funcionalidades importantes: ações sobre mensagens (edit, delete, react, read, presence, forward), publicação de Status/Stories, configurações avançadas de grupo (announce, locked, join-approval), download de mídia e importação de histórico Chatwoot. Isso força o usuário a recorrer ao Swagger ou a ferramentas externas (curl, Postman) para operações que deveriam estar acessíveis no painel.

## What Changes

- **Ações sobre mensagens**: adicionar no contexto de mensagens as operações de editar, deletar, reagir, marcar como lido, definir presença e encaminhar mensagens
- **Status/Stories**: nova seção para publicar Status de texto, imagem e vídeo
- **Configurações de grupo**: toggles para announce-only, locked e join-approval na aba de settings de grupo
- **Media page**: transformar o placeholder atual em galeria funcional que consome `GET /media/:messageId`
- **Chatwoot import**: botão de importação de histórico no card de configuração Chatwoot
- **Refatorações de UX**: extrair padrões duplicados (table UI, `wrapAction`, schemas) em composables/utils compartilhados; adicionar confirmações em ações destrutivas

## Não-objetivos

- Não será criado um chat em tempo real (conversação tipo WhatsApp Web) — o dashboard continua como painel de gestão/API
- Não serão alteradas rotas do backend Go — toda a mudança é exclusivamente frontend
- Não serão adicionados testes E2E nesta iteração (mas o código deve ser testável)
- Não será feita refatoração do layout/sidebar — a estrutura de navegação atual permanece

## Capabilities

### New Capabilities
- `message-actions`: Operações sobre mensagens existentes — edit, delete, react, mark-read, set-presence, forward
- `status-publishing`: Publicação de WhatsApp Status (Stories) — texto, imagem e vídeo
- `group-admin-settings`: Configurações administrativas de grupo — announce, locked, join-approval
- `media-gallery`: Galeria de mídia funcional com download via `GET /media/:messageId`
- `chatwoot-import`: Botão de importação de histórico no card Chatwoot
- `shared-ui-patterns`: Extração de padrões duplicados — table UI config, wrapAction, confirmação destrutiva

### Modified Capabilities
<!-- Nenhuma capability existente tem requisitos alterados -->

## Impact

- **Arquivos novos**: ~3-4 composables em `web/app/composables/`, 1-2 componentes em `web/app/components/sessions/`
- **Arquivos modificados**: `messages.vue`, `media.vue`, `groups.vue`, `contacts.vue`, `settings.vue`, `ChatwootConfigCard.vue`, `SendMessageModal.vue`, `group/SettingsTab.vue`
- **Composables modificados**: `useMessageSender.ts` (adicionar forward/status types)
- **Dependências**: nenhuma nova dependência npm necessária — tudo usa Nuxt UI existente
- **Backend**: zero alterações — todas as rotas já existem em `internal/server/router.go`

## Riscos e mitigações

| Risco | Mitigação |
|---|---|
| `SendMessageModal.vue` já tem 658 linhas e ficará maior com status/forward | Decompor em sub-components por tipo antes de adicionar features |
| Ações destrutivas (delete msg, delete chat) sem confirmação | Adicionar `DeleteModal` ou confirm dialog como parte de `shared-ui-patterns` |
| `GET /media/:messageId` pode retornar binário grande | Usar lazy loading, thumbnails e download progressivo |
| Forward de mensagens precisa de seleção de destinatário | Reutilizar componente de contatos/JID input existente |
