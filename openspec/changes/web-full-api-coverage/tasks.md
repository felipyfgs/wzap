## 1. Padrões compartilhados (shared-ui-patterns)

- [x] 1.1 Criar `web/app/composables/useTableUI.ts` com constante `TABLE_UI` extraída do objeto `ui` duplicado nas tabelas
- [x] 1.2 Substituir objeto `ui` inline em `pages/index.vue`, `pages/sessions/[id]/contacts.vue`, `pages/sessions/[id]/messages.vue`, `pages/webhooks.vue` pela constante `TABLE_UI`
- [x] 1.3 Criar `web/app/composables/useActionWrapper.ts` com função `wrapAction(fn, { success, error })` usando `useToast()`
- [x] 1.4 Substituir funções `wrapAction` inline em `contacts.vue` e `messages.vue` pelo composable compartilhado
- [x] 1.5 Criar `web/app/components/sessions/ConfirmModal.vue` com props `title`, `description`, `confirmLabel`, `confirmColor`, `icon` e eventos `confirm`/`cancel`
- [x] 1.6 Integrar `ConfirmModal` nas ações destrutivas existentes: delete session (`index.vue`), delete chat (`contacts.vue`, `messages.vue`), block contact (`contacts.vue`)

## 2. Decomposição do SendMessageModal

- [x] 2.1 Extrair formulário de texto para `components/sessions/message-forms/TextForm.vue`
- [x] 2.2 Extrair formulário de mídia (image/video/document/audio) para `components/sessions/message-forms/MediaForm.vue`
- [x] 2.3 Extrair formulário de contato para `components/sessions/message-forms/ContactForm.vue`
- [x] 2.4 Extrair formulário de localização para `components/sessions/message-forms/LocationForm.vue`
- [x] 2.5 Extrair formulário de link para `components/sessions/message-forms/LinkForm.vue`
- [x] 2.6 Extrair formulário de poll para `components/sessions/message-forms/PollForm.vue`
- [x] 2.7 Extrair formulário de sticker para `components/sessions/message-forms/StickerForm.vue`
- [x] 2.8 Extrair formulário de button para `components/sessions/message-forms/ButtonForm.vue`
- [x] 2.9 Extrair formulário de list para `components/sessions/message-forms/ListForm.vue`
- [x] 2.10 Refatorar `SendMessageModal.vue` para usar os sub-components via `<component :is="...">`
- [x] 2.11 Verificar que todos os 12 tipos de mensagem continuam funcionando após a decomposição

## 3. Status/Stories (status-publishing)

- [x] 3.1 Adicionar tipos `status-text`, `status-image`, `status-video` ao `useMessageSender.ts` com endpoints `status/text`, `status/image`, `status/video`
- [x] 3.2 Criar formulários `StatusTextForm.vue`, `StatusImageForm.vue`, `StatusVideoForm.vue` em `message-forms/`
- [x] 3.3 Adicionar os 3 tipos de Status no seletor de tipo do `SendMessageModal` com label "Status:" como prefixo
- [x] 3.4 Ocultar campo phone/JID quando tipo selecionado for `status-*` (auto-preencher com `status@broadcast`)
- [x] 3.5 Adicionar schemas Zod para os 3 tipos de Status (reutilizar textSchema e mediaSchema)

## 4. Ações sobre mensagens (message-actions)

- [x] 4.1 Criar `components/sessions/EditMessageModal.vue` com textarea para novo conteúdo
- [x] 4.2 Criar `components/sessions/ReactMessageModal.vue` com input de emoji
- [x] 4.3 Criar `components/sessions/ForwardMessageModal.vue` com input de telefone/JID destino
- [x] 4.4 Criar `components/sessions/PresenceModal.vue` com seletor de tipo (composing/recording/paused)
- [x] 4.5 Adicionar itens ao dropdown de ações em `messages.vue`: Edit (fromMe only), Delete (com ConfirmModal), React, Mark Read, Set Presence, Forward
- [x] 4.6 Conectar cada modal/ação aos endpoints: `POST /messages/edit`, `POST /messages/delete`, `POST /messages/reaction`, `POST /messages/read`, `POST /messages/presence`, `POST /messages/forward`

## 5. Configurações de grupo (group-admin-settings)

- [x] 5.1 Adicionar toggle "Announce Only" no `components/sessions/group/SettingsTab.vue` com `POST /groups/announce`
- [x] 5.2 Adicionar toggle "Locked" no `SettingsTab.vue` com `POST /groups/locked`
- [x] 5.3 Adicionar toggle "Join Approval" no `SettingsTab.vue` com `POST /groups/join-approval`
- [x] 5.4 Implementar rollback visual do switch em caso de erro na requisição

## 6. Galeria de mídia (media-gallery)

- [x] 6.1 Substituir placeholder em `pages/sessions/[id]/media.vue` por página funcional com fetch de mensagens
- [x] 6.2 Implementar filtro client-side por tipo de mídia (All, Images, Videos, Documents, Audio)
- [x] 6.3 Criar grid de cards com ícone/thumbnail por tipo e metadata (timestamp, chat, tipo)
- [x] 6.4 Implementar download via `GET /media/:messageId` usando link `<a download>`
- [x] 6.5 Implementar preview de imagens em modal (lightbox) ao clicar no thumbnail
- [x] 6.6 Adicionar toolbar com filtro de tipo e contador de resultados

## 7. Importação Chatwoot (chatwoot-import)

- [x] 7.1 Adicionar botão "Import History" no `ChatwootConfigCard.vue`, visível apenas quando `hasConfig: true`
- [x] 7.2 Integrar `ConfirmModal` no botão com descrição explicando que o processo pode demorar
- [x] 7.3 Conectar ao endpoint `POST /integrations/chatwoot/import` com feedback de loading e toast

## 8. Validação final

- [x] 8.1 Verificar manualmente todas as 16 rotas novas no dashboard (edit, delete, react, read, presence, forward, status x3, announce, locked, join-approval, media download, chatwoot import)
- [x] 8.2 Confirmar que nenhuma funcionalidade existente foi quebrada (12 tipos de mensagem, CRUD sessions, webhooks, contacts, groups, newsletters)
- [x] 8.3 Testar responsive em mobile (sidebar collapse + modais)
