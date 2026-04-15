## ADDED Requirements

### Requirement: Pagina dedicada de status
O sistema SHALL criar uma pagina `/sessions/:id/status` no frontend Nuxt com visual de stories. A pagina SHALL listar contatos que publicaram status (agrupados por `sender_jid`) com indicador visual de stories nao visualizados.

#### Scenario: Acesso a pagina de status
- **WHEN** o usuario navega para `/sessions/:id/status`
- **THEN** a pagina exibe uma lista de contatos com stories disponiveis, ordenados por timestamp do status mais recente

### Requirement: Card de story (StatusStoryCard)
O sistema SHALL criar um componente `StatusStoryCard` que exibe a foto do contato (ou avatar placeholder), nome, e timestamp do ultimo status. O card SHALL ter borda circular com gradiente colorido (similar ao WhatsApp) para indicar stories disponiveis.

#### Scenario: Exibicao do card
- **WHEN** um contato tem stories disponiveis
- **THEN** o card exibe avatar, nome e indicador visual de stories

### Requirement: Modal de visualizacao de status (StatusViewModal)
O sistema SHALL criar um modal `StatusViewModal` que exibe stories fullscreen com navegacao. O modal SHALL permitir avancar/retroceder entre stories do mesmo contato e fechar. Para status de imagem/video, SHALL exibir a media. Para status de texto, SHALL exibir o texto com fundo colorido.

#### Scenario: Visualizacao de story de imagem
- **WHEN** o usuario clica em um status de imagem
- **THEN** o modal abre fullscreen exibindo a imagem com caption

#### Scenario: Navegacao entre stories
- **WHEN** o usuario clica/toca na metade direita da tela
- **THEN** avanca para o proximo story do mesmo contato

#### Scenario: Fechamento do modal
- **WHEN** o usuario clica no botao de fechar ou pressiona Escape
- **THEN** o modal fecha e retorna a lista de status

### Requirement: Modal de envio de status (StatusSendModal)
O sistema SHALL criar um modal `StatusSendModal` dedicado ao envio de status, separado do `SendMessageModal`. O modal SHALL permitir escolher entre texto, imagem e video. O form de texto SHALL incluir campo de texto e opcoes de cor de fundo e fonte. Os forms de imagem e video SHALL permitir upload de arquivo ou URL.

#### Scenario: Envio de status de texto
- **WHEN** o usuario preenche o texto e clica em enviar
- **THEN** o status e enviado via API e o modal fecha com feedback de sucesso

#### Scenario: Envio de status de imagem
- **WHEN** o usuario seleciona uma imagem e clica em enviar
- **THEN** o status de imagem e enviado via API e o modal fecha

### Requirement: Composable useStatus
O sistema SHALL criar um composable `useStatus` em `web/app/composables/useStatus.ts` com as funcoes: `fetchStatuses(sessionId)` para listar status recebidos, `fetchContactStatuses(sessionId, senderJid)` para status de um contato, `sendStatusText(sessionId, payload)`, `sendStatusImage(sessionId, payload)`, `sendStatusVideo(sessionId, payload)` para envio.

#### Scenario: Buscar status
- **WHEN** `fetchStatuses` e chamado
- **THEN** retorna a lista de status do endpoint `GET /sessions/:id/status`

### Requirement: Link de navegacao na sidebar
O layout `default.vue` SHALL incluir um link "Status" no `sessionNavLinks` com icon `i-lucide-circle-dot` apontando para `/sessions/:id/status`.

#### Scenario: Link visivel
- **WHEN** o usuario esta em uma pagina de sessao
- **THEN** o link "Status" aparece na sidebar de navegacao

### Requirement: Remocao de status do SendMessageModal
O `SendMessageModal` SHALL remover as opcoes `status-text`, `status-image` e `status-video` do seletor de tipo de mensagem. O `useMessageSender` SHALL remover esses tipos do `MessageType`, `MESSAGE_TYPE_OPTIONS` e `TYPE_TO_ENDPOINT`.

#### Scenario: Sem opcoes de status no modal de mensagens
- **WHEN** o usuario abre o SendMessageModal
- **THEN** o dropdown de tipo nao inclui opcoes de status
