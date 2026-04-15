## Context

O dashboard Nuxt (`web/`) é uma SPA que consome a API REST do wzap via proxy Nitro (`server/api/[...].ts`). Atualmente cobre ~80% dos endpoints expostos em `internal/server/router.go`. A análise de gaps identificou 16 rotas sem cobertura no frontend, distribuídas em 5 domínios: ações sobre mensagens, Status/Stories, configurações de grupo, mídia e Chatwoot import.

O frontend segue padrões consistentes:
- **Layout**: `UDashboardPanel` > `#header` (navbar + toolbar) > `#body`
- **Data fetching**: `api()` do composable `useWzap` com `useToast()` para feedback
- **Overlays**: `UModal` para formulários, `USlideover` para detail views
- **Listas**: `UTable` com paginação client-side ou `UCard` grid
- **Ações**: `UDropdownMenu` com items agrupados por categoria

Existem padrões duplicados que serão consolidados como parte desta mudança.

## Goals / Non-Goals

**Goals:**
- Cobertura de 100% das rotas autenticadas do backend no dashboard
- Consolidar padrões de UI duplicados em composables/utils reutilizáveis
- Manter consistência visual e de interação com as páginas existentes
- Adicionar confirmações em ações destrutivas

**Non-Goals:**
- Chat em tempo real (WhatsApp Web-like)
- Alterações no backend Go
- Testes E2E
- Redesign de layout/sidebar

## Decisions

### D1: Ações de mensagem como dropdown no `messages.vue` existente

**Decisão**: Adicionar edit, delete, react, forward, mark-read e set-presence como itens no `UDropdownMenu` já existente em cada row da tabela de mensagens.

**Alternativas consideradas**:
- *Toolbar de ações com seleção múltipla*: mais complexo, não necessário para MVP
- *Página separada de detalhes de mensagem*: over-engineering para ações simples

**Razão**: segue o padrão já estabelecido em `contacts.vue` (actions dropdown por row). Cada ação abre um modal específico quando precisa de input (edit → modal com textarea, react → modal com emoji, forward → modal com seletor de destinatário).

### D2: Status/Stories como nova seção no `SendMessageModal`

**Decisão**: Adicionar `status-text`, `status-image` e `status-video` como novos tipos no `useMessageSender`, com campo `phone` omitido (Status vai para `status@broadcast`).

**Alternativas consideradas**:
- *Página separada `/sessions/:id/status`*: adicionaria mais um item na sidebar, mas o volume de UI é pequeno (3 formulários)
- *Modal separado*: duplicaria lógica de envio de mídia

**Razão**: O `SendMessageModal` já suporta 12 tipos com schemas Zod dinâmicos. Adicionar 3 tipos de Status segue o mesmo padrão. O phone field será auto-preenchido com `status@broadcast` e hidden quando o tipo for `status-*`.

**Pré-requisito**: decompor o `SendMessageModal` (658 linhas) em sub-components por tipo de mensagem antes de adicionar novos tipos.

### D3: Group settings como toggles no `SettingsTab.vue` existente

**Decisão**: Adicionar `USwitch` para announce, locked e join-approval no componente `group/SettingsTab.vue`, ao lado do toggle de ephemeral que já existe.

**Razão**: segue o padrão exato do ephemeral toggle. Cada switch faz `POST` para a rota correspondente (`/groups/announce`, `/groups/locked`, `/groups/join-approval`).

### D4: Media page com galeria lazy-load

**Decisão**: Substituir o placeholder de `media.vue` por uma página que lista mensagens com mídia (filtro do `GET /messages`) e permite download individual via `GET /media/:messageId`.

**Fluxo de dados**:
```
messages.vue (GET /messages)
    │
    ├─ filtra msgs com mediaUrl != null
    │
    └─ media.vue
         │
         ├─ grid de thumbnails (UCard)
         │
         └─ click → GET /media/:messageId → download/preview
```

**Alternativas consideradas**:
- *Endpoint dedicado de listagem de mídia*: não existe no backend, exigiria mudança Go
- *Inline na tabela de mensagens*: poluiria a UX de mensagens com previews

**Razão**: Reutiliza endpoints existentes sem alterar o backend. O `GET /messages` já retorna `mediaUrl` quando disponível.

### D5: Extração de padrões duplicados

**Decisão**: Criar os seguintes artefatos compartilhados:

| Artefato | Conteúdo |
|---|---|
| `composables/useTableUI.ts` | Constante com o objeto `ui` do UTable (duplicado em 4+ arquivos) |
| `composables/useActionWrapper.ts` | `wrapAction(fn, successMsg, errorMsg)` genérico (duplicado em contacts/messages) |
| `components/sessions/ConfirmModal.vue` | Modal de confirmação reutilizável para ações destrutivas |

### D6: Chatwoot import como botão no card existente

**Decisão**: Adicionar botão "Import History" no `ChatwootConfigCard.vue` que faz `POST /integrations/chatwoot/import`. Visível apenas quando a integração está configurada.

**Razão**: é uma ação simples (1 botão + confirmação + feedback). Não justifica componente separado.

## Riscos / Trade-offs

| Risco | Mitigação |
|---|---|
| Decomposição do `SendMessageModal` pode quebrar funcionalidade existente | Fazer como primeiro passo, testar manualmente todos os 12 tipos antes de adicionar novos |
| `GET /media/:messageId` retorna binário que pode ser grande | Usar `<a download>` para download direto, não carregar no DOM; para preview, usar URL temporária com `URL.createObjectURL` |
| Forward de mensagens precisa de messageId + destinatário | Criar `ForwardMessageModal` com input de JID e seleção da mensagem na row |
| Reações a mensagens: emoji picker pode adicionar dependência | Usar input de texto simples (emoji nativo do OS) ao invés de pacote de emoji picker |

## Open Questions

1. O endpoint `POST /messages/presence` define typing/recording para um chat — deve ficar como ação de mensagem ou como feature separada na toolbar?
2. O `GET /media/:messageId` retorna o binário direto ou um JSON com URL assinada? Precisa verificar o handler para definir o approach de download.
