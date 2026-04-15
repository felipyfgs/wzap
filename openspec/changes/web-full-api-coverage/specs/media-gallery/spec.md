## ADDED Requirements

### Requirement: Listar mensagens com mídia
O dashboard SHALL exibir na página Media uma lista de mensagens que possuem mídia associada, obtidas via `GET /sessions/:sessionId/messages`.

#### Scenario: Carregamento da galeria
- **WHEN** o usuário navega para `/sessions/:id/media`
- **THEN** o sistema faz `GET /messages?limit=200&offset=0` e filtra mensagens com mídia disponível (campo `mediaUrl` ou `hasMedia`)
- **THEN** exibe os resultados em grid de cards com tipo de mídia, timestamp e chat de origem

#### Scenario: Galeria vazia
- **WHEN** não existem mensagens com mídia
- **THEN** exibe empty state com ícone, texto "No media found" e sugestão para enviar mídia

#### Scenario: Loading state
- **WHEN** os dados estão sendo carregados
- **THEN** exibe spinner centralizado (padrão do projeto)

### Requirement: Download de mídia individual
O dashboard SHALL permitir download de mídia via `GET /sessions/:sessionId/media/:messageId`.

#### Scenario: Download bem-sucedido
- **WHEN** o usuário clica em "Download" em um item da galeria
- **THEN** o sistema faz `GET /media/:messageId` e inicia download do arquivo
- **THEN** o arquivo DEVE ser salvo com nome descritivo (tipo + timestamp)

#### Scenario: Erro no download
- **WHEN** a mídia não está mais disponível no servidor
- **THEN** o sistema exibe toast de erro "Media not available"

### Requirement: Preview de imagens
O dashboard SHALL exibir preview inline de mídias do tipo imagem.

#### Scenario: Preview de imagem
- **WHEN** o item da galeria é do tipo image (image/jpeg, image/png, image/webp)
- **THEN** o sistema DEVE exibir thumbnail da imagem no card
- **THEN** ao clicar na thumbnail, abre preview em tamanho maior (modal ou lightbox)

#### Scenario: Tipos não previewáveis
- **WHEN** o item é do tipo document, audio ou video
- **THEN** o card exibe ícone representativo do tipo de arquivo (sem preview inline)

### Requirement: Filtro por tipo de mídia
O dashboard SHALL permitir filtrar a galeria por tipo de mídia.

#### Scenario: Filtro por tipo
- **WHEN** o usuário seleciona um filtro (All, Images, Videos, Documents, Audio)
- **THEN** a galeria exibe apenas itens do tipo selecionado
- **THEN** o contador no toolbar DEVE refletir a contagem filtrada
