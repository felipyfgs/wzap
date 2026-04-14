## ADDED Requirements

### Requirement: Solicitar retry ao falhar download de mídia do history sync

Ao processar um `history_sync`, se o download de uma mídia falhar com erro HTTP 403, 404 ou 410, o sistema deve enviar uma solicitação de retry ao WhatsApp em vez de descartar silenciosamente.

#### Scenario: Download falha com 403 durante history sync
- **WHEN** `DownloadMediaByPath` retorna `ErrMediaDownloadFailedWith403`
- **THEN** `RequestMediaRetry` é chamado com o `MessageInfo` e `mediaKey` da mensagem
- **THEN** a entrada é adicionada ao cache de retry com TTL de 10 minutos
- **THEN** a mensagem é persistida normalmente (sem `media_url`)

#### Scenario: Download falha com 404 ou 410 durante history sync
- **WHEN** `DownloadMediaByPath` retorna `ErrMediaDownloadFailedWith404` ou `ErrMediaDownloadFailedWith410`
- **THEN** o mesmo comportamento do cenário 403 se aplica

#### Scenario: Download falha com outro erro (ex: timeout, rede)
- **WHEN** `DownloadMediaByPath` retorna erro diferente de 403/404/410
- **THEN** o sistema apenas loga e descarta (comportamento atual mantido)

---

### Requirement: Processar resposta do retry e persistir mídia

Ao receber um evento `*events.MediaRetry` do whatsmeow, o sistema deve tentar baixar e persistir a mídia recuperada.

#### Scenario: Retry bem-sucedido com mídia disponível no celular
- **WHEN** chega `*events.MediaRetry` com `messageID` presente no cache
- **AND** `DecryptMediaRetryNotification` retorna `MediaRetryNotification_SUCCESS`
- **THEN** o sistema baixa a mídia usando o novo `DirectPath`
- **THEN** faz upload para o MinIO com a mesma chave S3 determinística
- **THEN** atualiza `media_url` na tabela `wz_messages`
- **THEN** remove a entrada do cache de retry

#### Scenario: Mídia não disponível no celular
- **WHEN** chega `*events.MediaRetry` com código de erro 2 (`ErrMediaNotAvailableOnPhone`)
- **THEN** o sistema loga em nível warn e remove a entrada do cache
- **THEN** nenhuma atualização é feita no banco

#### Scenario: Retry recebido sem entrada no cache (expirada ou desconhecida)
- **WHEN** chega `*events.MediaRetry` com `messageID` ausente no cache
- **THEN** o sistema ignora silenciosamente o evento

#### Scenario: Entrada expirou (TTL de 10 minutos)
- **WHEN** chega `*events.MediaRetry` após 10 minutos da solicitação
- **THEN** a entrada não está mais no cache
- **THEN** o evento é ignorado silenciosamente
