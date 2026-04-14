## ADDED Requirements

### Requirement: Importação de mensagens históricas para o Chatwoot via API REST
O sistema SHALL implementar `importHistory` que lê mensagens do `wz_messages` com `source='history_sync'` e `imported_to_chatwoot_at IS NULL`, ordenadas por `timestamp ASC`, e cria as entidades correspondentes no Chatwoot via API REST.

#### Scenario: Importação de mensagem de texto do histórico
- **WHEN** uma mensagem de histórico com `msg_type='text'` e `imported_to_chatwoot_at IS NULL` é encontrada
- **THEN** o sistema encontra ou cria a conversa no Chatwoot via `findOrCreateConversation`, cria a mensagem via `client.CreateMessage` com `sourceID='WAID:{messageID}'`, e marca `imported_to_chatwoot_at=now()` na mensagem local

#### Scenario: Importação de mensagem com mídia do histórico
- **WHEN** uma mensagem de histórico com `media_url` preenchida e `imported_to_chatwoot_at IS NULL` é encontrada
- **THEN** o sistema faz download do MinIO via presigned URL, cria a mensagem no Chatwoot via `CreateMessageWithAttachment` com o conteúdo binário, e marca como importada

#### Scenario: Importação de múltiplas mensagens com rate limiting
- **WHEN** o import encontra N mensagens para importar
- **THEN** o sistema processa as mensagens respeitando o rate limiter de 10 mensagens por segundo (100ms entre cada), atualizando a métrica `cw_history_import_progress` como porcentagem de progresso

#### Scenario: Importação parcial por erro
- **WHEN** o import falha no meio do processamento (ex: Chatwoot indisponível)
- **THEN** as mensagens já processadas ficam com `imported_to_chatwoot_at` preenchido. Uma re-execução retoma de onde parou, processando apenas mensagens não importadas.

#### Scenario: Import concorrente para mesma sessão
- **WHEN** duas chamadas de import são feitas simultaneamente para o mesmo `sessionID`
- **THEN** o sistema usa `singleflight.Group` para garantir que apenas uma execução corre por sessão, e a segunda chamada aguarda o resultado da primeira

### Requirement: Query de mensagens históricas não importadas
O `MessageRepo` SHALL fornecer método `FindUnimportedHistory(ctx, sessionID, since, limit, offset)` que retorna mensagens onde `source='history_sync'`, `imported_to_chatwoot_at IS NULL`, `timestamp >= since`, ordenadas por `timestamp ASC, id ASC`, com paginação via `limit` e `offset`.

#### Scenario: Busca paginada de mensagens não importadas
- **WHEN** `FindUnimportedHistory` é chamado com `sessionID`, `since=7d atrás`, `limit=100`, `offset=0`
- **THEN** retorna até 100 mensagens de histórico não importadas dentro do período, ordenadas cronologicamente

#### Scenario: Nenhuma mensagem para importar
- **WHEN** não existem mensagens com `source='history_sync'` e `imported_to_chatwoot_at IS NULL` no período
- **THEN** retorna slice vazio (não nil) e nenhum erro

### Requirement: Marcação de mensagem como importada no Chatwoot
O `MessageRepo` SHALL fornecer método `MarkImportedToChatwoot(ctx, sessionID, msgID string)` que seta `imported_to_chatwoot_at=now()` para a mensagem.

#### Scenario: Marcar mensagem importada com sucesso
- **WHEN** uma mensagem é criada com sucesso no Chatwoot
- **THEN** `MarkImportedToChatwoot` seta `imported_to_chatwoot_at` para o timestamp atual

### Requirement: Endpoint manual de importação de histórico
O handler `ImportHistory` SHALL aceitar `POST /sessions/{sessionId}/integrations/chatwoot/import` com body `{period, customDays?}`, retornar `202 Accepted` imediatamente, e executar o import em background.

#### Scenario: Trigger manual de import via API
- **WHEN** `POST /sessions/{id}/integrations/chatwoot/import` com `{period: "7d"}` é recebido
- **THEN** retorna `202 Accepted` com `{success: true, data: {status: "importing", period: "7d"}}` e inicia import em goroutine

#### Scenario: Trigger manual com período customizado
- **WHEN** `POST /sessions/{id}/integrations/chatwoot/import` com `{period: "custom", customDays: 15}` é recebido
- **THEN** retorna `202 Accepted` e importa mensagens dos últimos 15 dias

#### Scenario: Trigger com sessão não configurada
- **WHEN** o endpoint recebe request para sessão sem configuração Chatwoot
- **THEN** retorna `404` com mensagem "Chatwoot integration not configured for this session"

#### Scenario: Import automático ao conectar
- **WHEN** `handleConnected` é chamado e `cfg.ImportOnConnect=true`
- **THEN** o sistema dispara `importHistory` em goroutine separada com o período configurado em `cfg.ImportPeriod`
