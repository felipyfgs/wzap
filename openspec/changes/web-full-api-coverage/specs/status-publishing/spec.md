## ADDED Requirements

### Requirement: Publicar Status de texto
O dashboard SHALL permitir publicar um Status (Story) de texto via `POST /sessions/:sessionId/messages/status/text`.

#### Scenario: Publicação de texto bem-sucedida
- **WHEN** o usuário seleciona o tipo "Status Text" no SendMessageModal
- **THEN** o campo de telefone/JID DEVE ser automaticamente preenchido com `status@broadcast` e ocultado da interface
- **THEN** o formulário exibe apenas o campo de texto (body)
- **THEN** ao submeter, faz `POST /messages/status/text` com `{ body }` e exibe toast de sucesso

### Requirement: Publicar Status de imagem
O dashboard SHALL permitir publicar um Status de imagem via `POST /sessions/:sessionId/messages/status/image`.

#### Scenario: Publicação de imagem bem-sucedida
- **WHEN** o usuário seleciona o tipo "Status Image" no SendMessageModal
- **THEN** o campo de telefone/JID DEVE ser ocultado (auto-preenchido com `status@broadcast`)
- **THEN** o formulário exibe campos de upload de imagem (file ou URL), mimeType e caption
- **THEN** ao submeter, faz `POST /messages/status/image` com `{ mimeType, base64|url, caption }` e exibe toast de sucesso

#### Scenario: Validação de mídia
- **WHEN** o usuário tenta submeter sem arquivo e sem URL
- **THEN** o formulário DEVE exibir erro de validação "Provide a file or a URL"

### Requirement: Publicar Status de vídeo
O dashboard SHALL permitir publicar um Status de vídeo via `POST /sessions/:sessionId/messages/status/video`.

#### Scenario: Publicação de vídeo bem-sucedida
- **WHEN** o usuário seleciona o tipo "Status Video" no SendMessageModal
- **THEN** o campo de telefone/JID DEVE ser ocultado (auto-preenchido com `status@broadcast`)
- **THEN** o formulário exibe campos de upload de vídeo (file ou URL), mimeType e caption
- **THEN** ao submeter, faz `POST /messages/status/video` com `{ mimeType, base64|url, caption }` e exibe toast de sucesso

### Requirement: Tipos de Status aparecem no seletor de tipo de mensagem
O seletor de tipo no `SendMessageModal` DEVE incluir as opções "Status Text", "Status Image" e "Status Video" separadas dos tipos de mensagem direta.

#### Scenario: Seletor de tipo com Status
- **WHEN** o usuário abre o SendMessageModal
- **THEN** o seletor de tipo DEVE exibir todos os 12 tipos existentes mais os 3 tipos de Status
- **THEN** os tipos de Status DEVEM aparecer agrupados ou com label indicativo (ex: prefixo "Status:")
