## ADDED Requirements

### Requirement: Upload de mídia inbound para MinIO

O sistema DEVE armazenar mídia recebida do WhatsApp no MinIO para servir ao Chatwoot no modo cloud.

#### Scenario: Upload bem-sucedido de imagem
- **WHEN** uma imagem chega do WhatsApp no modo cloud
- **THEN** o sistema DEVE:
  1. Baixar via `mediaDownloader.DownloadMediaByPath()`
  2. Upload para MinIO com chave `chatwoot/{sessionID}/{msgID}/{filename}`
  3. Obter URL pré-assinada via `mediaPresigner.GetPresignedURL()`
- **THEN** a URL DEVE ser usada no campo `link` do payload Cloud API

#### Scenario: Upload para diferentes tipos de mídia
- **WHEN** mídia de tipo `image`, `video`, `audio`, `document` ou `sticker` chega no modo cloud
- **THEN** todas DEVEM seguir o mesmo fluxo de upload para MinIO
- **THEN** o filename DEVE ser derivado do `msgID` + extensão baseada no mime_type quando não disponível

#### Scenario: MinIO não configurado
- **WHEN** MinIO não está configurado (`s.mediaPresigner == nil`)
- **THEN** o sistema DEVE logar warning `"MinIO not configured, cannot upload media for cloud mode"`
- **THEN** a mensagem DEVE ser enviada ao Chatwoot apenas como texto (caption sem mídia)

#### Scenario: Filename com caracteres especiais
- **WHEN** o filename original contém caracteres especiais ou espaços
- **THEN** o sistema DEVE sanitizar o filename para uso como chave MinIO (substituir espaços por `_`, remover caracteres problemáticos)

---

### Requirement: Endpoint de consulta de mídia simulando Graph API

O sistema DEVE expor `GET /:version/:phone/:media_id` que retorna metadados de mídia no formato Cloud API.

#### Scenario: Consulta de mídia existente
- **WHEN** Chatwoot faz `GET /v20.0/5511888888888/chatwoot%2Fsession1%2Fmsg123%2Fphoto.jpg`
- **THEN** o sistema DEVE:
  1. Decodificar o `media_id` (URL-encoded) para obter a chave MinIO
  2. Gerar URL pré-assinada via `mediaPresigner.GetPresignedURL()`
- **THEN** retornar HTTP 200 com:
  ```json
  {
    "url": "<presigned_url>",
    "mime_type": "<mime_type>",
    "sha256": "<hash>",
    "file_size": <bytes>,
    "id": "<media_id>",
    "messaging_product": "whatsapp"
  }
  ```

#### Scenario: Mídia não encontrada
- **WHEN** o `media_id` não existe no MinIO
- **THEN** retornar HTTP 404 com:
  ```json
  {
    "error": {
      "message": "Media not found",
      "type": "OAuthException",
      "code": 100
    }
  }
  ```

#### Scenario: Autenticação do endpoint de mídia
- **WHEN** a requisição não contém Bearer token válido
- **THEN** retornar HTTP 401 com erro Cloud API format

---

### Requirement: Download de mídia outbound (Chatwoot → WA)

O sistema DEVE suportar download de mídia enviada pelo Chatwoot para encaminhar ao WhatsApp.

#### Scenario: Mídia via link direto
- **WHEN** Chatwoot envia `image.link: "https://chatwoot.example.com/rails/active_storage/blobs/xxx/photo.jpg"`
- **THEN** o sistema DEVE baixar a imagem da URL via HTTP GET
- **THEN** passar os bytes para `messageSvc.SendImage()` com a URL original

#### Scenario: Mídia via media_id
- **WHEN** Chatwoot envia `image.id: "media_123"` sem `link`
- **THEN** o sistema DEVE resolver o `media_id` no MinIO para obter a URL
- **THEN** usar a URL para enviar via `messageSvc.SendImage()`

#### Scenario: Falha no download da mídia
- **WHEN** o download da URL falha (timeout, 404, etc.)
- **THEN** o sistema DEVE retornar HTTP 400 com erro Cloud API:
  ```json
  {
    "error": {
      "message": "Failed to download media",
      "type": "OAuthException",
      "code": 131053
    }
  }
  ```

---

### Requirement: TTL e limpeza de mídia no MinIO

A mídia armazenada no MinIO para o modo cloud DEVE ter política de expiração.

#### Scenario: Mídia expirada
- **WHEN** uma mídia foi armazenada há mais de 7 dias (TTL padrão)
- **THEN** o MinIO DEVE removê-la automaticamente via lifecycle policy do bucket
- **THEN** consultas posteriores ao `media_id` DEVEM retornar 404

#### Scenario: Chatwoot busca mídia antes da expiração
- **WHEN** Chatwoot recebe o webhook com `link` e faz download imediatamente
- **THEN** o download DEVE funcionar normalmente (URL pré-assinada válida)
- **THEN** o TTL de 7 dias é margem de segurança suficiente (Chatwoot baixa na hora)
