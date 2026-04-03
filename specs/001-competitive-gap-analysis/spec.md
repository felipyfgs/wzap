# Feature Specification: wzap Competitive Gap Analysis & Feature Roadmap

**Feature Branch**: `001-competitive-gap-analysis`
**Created**: 2026-04-03
**Status**: Draft
**Input**: Analise do wzap vs mercado de APIs WhatsApp baseadas em whatsmeow/Baileys — identificar gaps e oportunidades

## Analise Competitiva

### Concorrentes Diretos

| Projeto | Stars | Linguagem | Engine | Licenca |
|---------|-------|-----------|--------|---------|
| Evolution API | 7.7k | TypeScript | Baileys | Apache 2.0 |
| CodeChat | 1.2k | TypeScript | Baileys | Apache 2.0 |
| WPPConnect | 3.3k | TypeScript | wa-js/Puppeteer | LGPL 3.0 |
| **wzap** | — | **Go** | **whatsmeow** | — |

### Matriz de Features

| Feature | Evolution API | CodeChat | WPPConnect | wzap | Gap? |
|---------|:---:|:---:|:---:|:---:|:---:|
| **Sessoes/Instancias** | | | | | |
| Create/Delete instancias | Sim | Sim | N/A | Sim | — |
| Multi-instancia | Sim | Sim | Sim | Sim | — |
| Connect/Disconnect/Reconnect | Sim | Sim | Sim | Sim | — |
| QR Code | Sim | Sim | Sim | Sim | — |
| Pairing code | Sim | Sim | Sim | Sim | — |
| Restart instancia | Sim | Nao | N/A | Nao | GAP |
| **Tipos de Mensagem** | | | | | |
| Texto | Sim | Sim | Sim | Sim | — |
| Imagem/Video/Audio/Doc | Sim | Sim | Sim | Sim | — |
| Contact vCard | Sim | Sim | Sim | Sim | — |
| Localizacao | Sim | Sim | Sim | Sim | — |
| Sticker | Sim | Nao | Sim | Sim | — |
| Enquete | Sim | Nao | Sim | Sim | — |
| Reacao (emoji) | Sim | Sim | Sim | Sim | — |
| Botoes interativos | Cloud API | Nao | Sim | Sim | — |
| Lista interativa | Staging | Nao | Sim | Sim | — |
| Preview de link | Sim | Sim | Sim | Sim | — |
| Status/Stories | Sim | Nao | Sim | Nao | GAP |
| Mensagem de voz (PTT) | Sim | Sim | Sim | Sim | — |
| Editar mensagem | Nao | Nao | Nao | Sim | VANTAGEM |
| Deletar mensagem | Sim | Nao | Sim | Sim | — |
| Encaminhar mensagem | Sim | Nao | Sim | Nao | GAP |
| Responder/Citar | Sim | Nao | Sim | Sim | — |
| Mencionar (@user) | Sim | Nao | Sim | Nao | GAP |
| **Operacoes de Chat** | | | | | |
| Marcar como lido | Sim | Nao | Sim | Sim | — |
| Marcar como nao lido | Sim | Nao | Sim | Nao | GAP |
| Presenca (digitando) | Sim | Nao | Sim | Sim | — |
| Arquivar chat | Sim | Nao | Sim | Sim | — |
| Fixar/Desfixar chat | Nao | Nao | Sim | Sim | — |
| Silenciar chat | Nao | Nao | Sim | Sim | — |
| Bloquear/Desbloquear | Sim | Nao | Sim | Sim | — |
| **Grupos** | | | | | |
| Criar grupo | Sim | Nao | Sim | Sim | — |
| Listar grupos | Sim | Sim | Sim | Sim | — |
| Atualizar nome/descricao/foto | Sim | Nao | Sim | Sim | — |
| Add/remove/promote/demote | Sim | Nao | Sim | Sim | — |
| Sair do grupo | Sim | Nao | Sim | Sim | — |
| Link de convite | Sim | Nao | Sim | Sim | — |
| Revogar convite | Sim | Nao | Sim | Nao | GAP |
| Entrar via link | Sim | Nao | Sim | Sim | — |
| Mensagens efemeras | Sim | Nao | Sim | Sim | — |
| Aprovacao de membros | Nao | Nao | Nao | Sim | VANTAGEM |
| **Contatos** | | | | | |
| Listar contatos | Sim | Sim | Sim | Sim | — |
| Verificar numero | Sim | Nao | Sim | Sim | — |
| Foto de perfil | Sim | Nao | Sim | Sim | — |
| Info do usuario | Nao | Nao | Sim | Sim | — |
| Lista de bloqueados | Nao | Nao | Nao | Sim | VANTAGEM |
| **Perfil** | | | | | |
| Atualizar nome perfil | Sim | Nao | Sim | Nao | GAP |
| Atualizar foto perfil | Sim | Nao | Sim | Sim | — |
| Atualizar status/sobre | Sim | Nao | Sim | Sim | — |
| **Labels/Tags** | | | | | |
| Gerenciar labels | Nao | Sim | Nao | Sim | VANTAGEM |
| Label em chats | Nao | Sim | Nao | Sim | VANTAGEM |
| **Newsletter/Canal** | | | | | |
| Criar newsletter | Nao | Nao | Nao | Sim | VANTAGEM |
| Subscribe/Unsubscribe | Nao | Nao | Nao | Sim | VANTAGEM |
| **Comunidade** | | | | | |
| Criar comunidade | Nao | Nao | Nao | Sim | VANTAGEM |
| **Webhooks/Eventos** | | | | | |
| Webhooks por instancia | Sim | Sim | N/A | Sim | — |
| URL por tipo de evento | Sim | Nao | N/A | Nao | GAP |
| WebSocket | Sim | Sim | N/A | Sim | — |
| **Integracoes** | | | | | |
| Chatwoot | Sim | Nao | Nao | Nao | GAP |
| Typebot | Sim | Nao | Nao | Nao | GAP |
| OpenAI | Sim | Nao | Nao | Nao | GAP |
| **Midia** | | | | | |
| Enviar via URL/base64 | Sim | Sim | Sim | Sim | — |
| Storage S3/MinIO | Sim | Nao | Nao | Sim | — |
| Conversao de audio (ffmpeg) | Sim | Nao | Nao | Nao | GAP |
| **Observabilidade** | | | | | |
| Prometheus metrics | Sim | Nao | N/A | Nao | GAP |
| Grafana dashboard | Sim | Nao | N/A | Nao | GAP |

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Envio de Status/Stories (Priority: P1)

Como usuario do wzap, quero enviar mensagens de status (Stories) do WhatsApp para meus contatos, permitindo comunicacao broadcast visual com duracao de 24h.

**Why this priority**: Status/Stories e uma feature presente no Evolution API e WPPConnect. E uma funcionalidade nativa muito usada do WhatsApp para marketing e comunicacao em massa.

**Independent Test**: Pode ser testada com `POST /sessions/:sessionId/messages/status` enviando texto, imagem ou video. Retorna o ID da mensagem de status.

**Acceptance Scenarios**:

1. **Given** uma sessao conectada, **When** envio um status de texto via API, **Then** o status e publicado e visivel para meus contatos por 24h
2. **Given** uma sessao conectada, **When** envio um status com imagem e legenda, **Then** o status com imagem e publicado corretamente

---

### User Story 2 - Encaminhar Mensagem (Priority: P1)

Como usuario do wzap, quero encaminhar uma mensagem existente para outro chat ou grupo, mantendo ou nao a referencia original.

**Why this priority**: Encaminhar mensagens e uma operacao basica do WhatsApp presente no Evolution API e WPPConnect. Ausencia disso e um gap funcional significativo.

**Independent Test**: `POST /sessions/:sessionId/messages/forward` com messageID e destino. A mensagem aparece no chat de destino.

**Acceptance Scenarios**:

1. **Given** uma mensagem recebida ou enviada, **When** encaminho para outro chat, **Then** a mensagem aparece no chat de destino com indicador de encaminhada
2. **Given** uma mensagem com midia, **When** encaminho, **Then** a midia e transferida junto

---

### User Story 3 - Mencionar Usuarios em Mensagens (Priority: P2)

Como usuario do wzap, quero mencionar (@user) contatos em mensagens de texto e grupos, chamando a atencao do usuario mencionado.

**Why this priority**: Mencoes sao essenciais para grupos e bots de atendimento. Presente no Evolution API e WPPConnect.

**Independent Test**: `POST /sessions/:sessionId/messages/text` com campo `mentionedJids` inclui as mencoes na mensagem.

**Acceptance Scenarios**:

1. **Given** um chat ou grupo, **When** envio mensagem mencionando um contato, **Then** o contato recebe notificacao e a mensagem mostra @nome

---

### User Story 4 - Nome de Perfil Atualizavel (Priority: P2)

Como usuario do wzap, quero atualizar meu nome de perfil (push name) diretamente pela API.

**Why this priority**: Presente no Evolution API e WPPConnect. Gap simples de implementar via whatsmeow.

**Independent Test**: `POST /sessions/:sessionId/profile/name` atualiza o nome do perfil.

**Acceptance Scenarios**:

1. **Given** uma sessao conectada, **When** atualizo o nome do perfil via API, **Then** o nome aparece atualizado para todos os contatos

---

### User Story 5 - Revogar Link de Convite de Grupo (Priority: P2)

Como admin de grupo, quero revogar o link de convite ativo, invalidando o link anterior e impedindo novos acessos nao autorizados.

**Why this priority**: Seguranca de grupos. Presente no Evolution API.

**Independent Test**: `POST /sessions/:sessionId/groups/invite-link/revoke` gera novo link.

**Acceptance Scenarios**:

1. **Given** um grupo que sou admin, **When** revogo o link de convite, **Then** o link anterior para de funcionar e um novo e gerado

---

### User Story 6 - Webhook por Tipo de Evento (Priority: P2)

Como desenvolvedor integrando com wzap, quero configurar URLs de webhook diferentes para cada tipo de evento, permitindo arquitetura de microsservicos.

**Why this priority**: Evolution API suporta isso. Facilita integracao com sistemas especializados.

**Independent Test**: Criar webhook com mapa de evento -> URL. Apenas os eventos configurados vao pra URL correta.

**Acceptance Scenarios**:

1. **Given** um webhook configurado, **When** recebo uma mensagem, **Then** o evento vai pra URL configurada para Message
2. **Given** URLs diferentes para Message e GroupInfo, **When** ambos eventos ocorrem, **Then** cada evento vai pra sua URL

---

### User Story 7 - Conversao de Audio (ffmpeg) (Priority: P3)

Como usuario do wzap, quero enviar audios em qualquer formato (MP3, WAV, OGG) e ter conversao automatica para o formato correto do WhatsApp (OGG Opus).

**Why this priority**: Presente no Evolution API. Melhora significativamente a DX.

**Independent Test**: Enviar MP3 via `/messages/audio` e receber como mensagem de voz reproduzivel.

**Acceptance Scenarios**:

1. **Given** um arquivo MP3, **When** envio via API de audio, **Then** o audio e convertido para OGG Opus e enviado como mensagem de voz
2. **Given** um arquivo OGG ja no formato correto, **When** envio via API, **Then** o audio e enviado sem conversao

---

### User Story 8 - Observabilidade com Prometheus (Priority: P3)

Como operador do wzap em producao, quero metricas em formato Prometheus para monitorar sessoes ativas, mensagens enviadas/recebidas, latencia de webhooks e erros.

**Why this priority**: Presente no Evolution API com Grafana dashboard. Essencial para operacao em producao.

**Independent Test**: Acessar `GET /metrics` e ver metricas no formato Prometheus.

**Acceptance Scenarios**:

1. **Given** o wzap rodando, **When** acesso `/metrics`, **Then** vejo metricas de sessoes conectadas, mensagens processadas e taxa de erro de webhooks

---

### User Story 9 - Marcar Chat como Nao Lido (Priority: P3)

Como usuario do wzap, quero marcar um chat como nao lido para revisitar depois.

**Why this priority**: Feature basica do WhatsApp presente no Evolution API. Gap simples.

**Independent Test**: `POST /sessions/:sessionId/chat/unread` marca o chat.

**Acceptance Scenarios**:

1. **Given** um chat com mensagens lidas, **When** marco como nao lido via API, **Then** o chat aparece como nao lido no WhatsApp

---

### Edge Cases

- O que acontece ao enviar status para uma sessao que nao esta conectada?
- Como o sistema lida com mencoes em grupos onde o usuario mencionado nao e membro?
- O que acontece ao encaminhar uma mensagem que foi deletada?
- Como lidar com conversao de audio quando ffmpeg nao esta disponivel no container?
- O que acontece ao revogar link de convite quando nao sou admin do grupo?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Sistema DEVE suportar envio de Status/Stories do WhatsApp (texto, imagem, video)
- **FR-002**: Sistema DEVE suportar encaminhamento de mensagens entre chats e grupos
- **FR-003**: Sistema DEVE suportar mencoes (@user) em mensagens de texto
- **FR-004**: Sistema DEVE permitir atualizacao do nome de perfil pela API
- **FR-005**: Sistema DEVE suportar revogacao de link de convite de grupo
- **FR-006**: Sistema DEVE suportar configuracao de URL de webhook por tipo de evento
- **FR-007**: Sistema DEVE converter automaticamente audios para OGG Opus via ffmpeg
- **FR-008**: Sistema DEVE fornecer endpoint de metricas no formato Prometheus
- **FR-009**: Sistema DEVE suportar marcacao de chat como nao lido
- **FR-010**: Sistema DEVE suportar restart de instancia (reconectar sem perder estado)

### Key Entities

- **Status Message**: Mensagem de status com tipo (texto/imagem/video), conteudo, duracao de 24h
- **Forwarded Message**: Referencia a mensagem original + chat de destino
- **Mention**: Lista de JIDs mencionados na mensagem
- **Event Webhook Mapping**: Mapa de tipo de evento -> URL de webhook
- **Prometheus Metrics**: Contadores e gauges para sessoes, mensagens, latencia, erros

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Todas as 10 features gap identificadas sao implementadas e testadas
- **SC-002**: wzap possui cobertura de features igual ou superior ao Evolution API em mensagens e grupos
- **SC-003**: Feature parity com Evolution API em pelo menos 90% das features de mensagens e grupos
- **SC-004**: Endpoint `/metrics` exporta pelo menos 15 metricas relevantes no formato Prometheus
- **SC-005**: Conversao de audio suporta pelo menos 3 formatos de entrada (MP3, WAV, M4A)

## wzap Vantagens Competitivas (manter/reforcar)

| Vantagem | Descricao |
|----------|-----------|
| Performance Go | Footprint de memoria/CPU significativamente menor que Node.js |
| Newsletter/Canal | Unico no mercado com suporte completo a newsletters |
| Comunidade | Unico com suporte a comunidades WhatsApp |
| Editar mensagem | Unico com suporte a edicao de mensagens |
| Aprovacao de membros | Features avancadas de grupo |
| Labels/Tags | Sistema completo de labels em chats e mensagens |
| NATS JetStream | Event streaming robusto e leve |
| MinIO nativo | Storage S3 integrado para midia |
| Rate limiting | Protecao contra abuso integrada |
| API key por sessao | Seguranca multi-tenant nativa |

## Assumptions

- O whatsmeow suporta envio de Status/Stories via protocolo WhatsApp Web (a ser verificado na fase de planejamento)
- ffmpeg pode ser adicionado ao container Docker sem impacto significativo no tamanho da imagem
- O endpoint Prometheus sera read-only, sem autenticacao (metrics internas)
- O roteamento de webhook por evento sera uma extensao do sistema de webhooks atual
