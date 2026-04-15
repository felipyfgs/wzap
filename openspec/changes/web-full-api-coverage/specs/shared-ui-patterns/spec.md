## ADDED Requirements

### Requirement: Constante de UI compartilhada para UTable
O projeto SHALL ter uma constante reutilizável com a configuração de estilo do `UTable` usada em todas as páginas.

#### Scenario: Uso em novas tabelas
- **WHEN** um desenvolvedor cria uma nova página com `UTable`
- **THEN** DEVE importar `TABLE_UI` de `utils/` ou `composables/useTableUI` em vez de duplicar o objeto de estilos

#### Scenario: Substituição nas tabelas existentes
- **WHEN** as tabelas existentes em `contacts.vue`, `messages.vue`, `index.vue` e `webhooks.vue` são refatoradas
- **THEN** o objeto `ui` inline DEVE ser substituído pela constante compartilhada
- **THEN** o comportamento visual DEVE permanecer idêntico

### Requirement: Composable de ação com feedback
O projeto SHALL ter um composable `useActionWrapper` que encapsula a lógica de try/catch/toast usada em ações assíncronas.

#### Scenario: Uso do wrapAction
- **WHEN** uma ação assíncrona é executada (ex: archive chat, delete, block)
- **THEN** o desenvolvedor DEVE usar `wrapAction(fn, { success, error })` do composable
- **THEN** em caso de sucesso, exibe toast com título de sucesso
- **THEN** em caso de erro, exibe toast com título de erro

#### Scenario: Substituição nos arquivos existentes
- **WHEN** as funções `wrapAction` inline em `contacts.vue` e `messages.vue` são refatoradas
- **THEN** DEVEM ser substituídas pelo composable compartilhado
- **THEN** o comportamento DEVE permanecer idêntico

### Requirement: Modal de confirmação reutilizável
O projeto SHALL ter um componente `ConfirmModal` para ações destrutivas.

#### Scenario: Confirmação de deleção
- **WHEN** o usuário tenta executar uma ação destrutiva (delete session, delete chat, block contact, delete message)
- **THEN** o sistema DEVE exibir um `ConfirmModal` com título, descrição e botões "Cancel" / "Confirm"
- **THEN** o botão "Confirm" DEVE ter `color: 'error'` para ações destrutivas
- **THEN** a ação só é executada após confirmação

#### Scenario: Modal customizável
- **WHEN** o `ConfirmModal` é usado
- **THEN** DEVE aceitar props: `title`, `description`, `confirmLabel`, `confirmColor`, `icon`
- **THEN** DEVE emitir evento `confirm` quando confirmado e `cancel` quando cancelado

#### Scenario: Loading state no confirm
- **WHEN** o usuário clica em "Confirm" e a ação está em andamento
- **THEN** o botão "Confirm" DEVE exibir estado loading e ficar desabilitado até a conclusão
