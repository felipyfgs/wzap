## ADDED Requirements

### Requirement: Download de mídias do histórico durante PersistHistorySync
O sistema SHALL fazer download e armazenamento de mídias (image, video, audio, document, sticker) recebidas via eventos `HistorySync` no momento em que `PersistHistorySync` processa cada mensagem. O download SHALL ser feito via `wa.Manager.DownloadMediaByPath` usando os campos `directPath`, `encFileHash`, `fileHash`, `mediaKey` e `fileLength` do proto `waE2E.Message`. O arquivo SHALL ser armazenado no MinIO com chave `{sessionID}/{messageID}`.

#### Scenario: Mensagem de histórico com imagem
- **WHEN** uma mensagem de HistorySync contém `ImageMessage` com `directPath` preenchido
- **THEN** o sistema faz download dos bytes via whatsmeow, faz upload para MinIO, e salva a presigned URL no campo `media_url` de `wz_messages`

#### Scenario: Mensagem de histórico com vídeo
- **WHEN** uma mensagem de HistorySync contém `VideoMessage` com `directPath` preenchido
- **THEN** o sistema faz download, upload para MinIO, e salva a presigned URL em `media_url`

#### Scenario: Mensagem de histórico com áudio
- **WHEN** uma mensagem de HistorySync contém `AudioMessage` com `directPath` preenchido
- **THEN** o sistema faz download, upload para MinIO, e salva a presigned URL em `media_url`

#### Scenario: Mensagem de histórico com documento
- **WHEN** uma mensagem de HistorySync contém `DocumentMessage` com `directPath` preenchido
- **THEN** o sistema faz download, upload para MinIO, e salva a presigned URL em `media_url`

#### Scenario: Mensagem de histórico com sticker
- **WHEN** uma mensagem de HistorySync contém `StickerMessage` com `directPath` preenchido
- **THEN** o sistema faz download, upload para MinIO, e salva a presigned URL em `media_url`

#### Scenario: Falha no download de mídia expirada
- **WHEN** o download falha (media key expirada ou erro de rede)
- **THEN** o sistema SHALL logar um Warn com sessionID e messageID, e continuar processamento sem bloquear a persistência da mensagem de texto. O campo `media_url` permanece vazio.

#### Scenario: Mídia já existente no MinIO
- **WHEN** uma mensagem de histórico já possui `media_url` preenchida (re-processamento)
- **THEN** o sistema SHALL pular o download e manter a URL existente

#### Scenario: Mensagem de histórico sem mídia
- **WHEN** uma mensagem de HistorySync contém apenas texto (Conversation ou ExtendedTextMessage)
- **THEN** o sistema não tenta download e persiste apenas o texto normalmente

### Requirement: Extração de informações de download do proto Message
O sistema SHALL extrair campos de download (`directPath`, `encFileHash`, `fileHash`, `mediaKey`, `fileLength`) do proto `waE2E.Message` para cada tipo de mídia suportado. A função `extractMediaDownloadInfo` SHALL retornar esses campos ou indicar que não há mídia para download.

#### Scenario: Extração de campos de ImageMessage
- **WHEN** o proto Message contém `ImageMessage`
- **THEN** a função retorna `directPath`, `encFileHash`, `fileHash`, `mediaKey`, `fileLength` do `ImageMessage` e `ok=true`

#### Scenario: Proto sem mídia
- **WHEN** o proto Message contém apenas `Conversation` ou `ExtendedTextMessage`
- **THEN** a função retorna `ok=false`
