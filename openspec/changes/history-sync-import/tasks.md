## 1. Download de Mídias do Histórico

- [ ] 1.1 Criar função `extractMediaDownloadInfo(msg *waE2E.Message) (directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int, ok bool)` em `internal/service/history.go` para extrair campos de download de ImageMessage, VideoMessage, AudioMessage, DocumentMessage e StickerMessage
- [ ] 1.2 Adicionar interface `MediaStorage` com método `UploadAndPresign(ctx, sessionID, messageID string, data []byte, mimeType string) (string, error)` e implementar wrapper sobre `storage.Minio`
- [ ] 1.3 Adicionar dependências `MediaDownloader` e `MediaStorage` ao `HistoryService` via construtor `NewHistoryService`
- [ ] 1.4 Implementar lógica de download + upload em `PersistHistorySync`: após `buildHistoryMessage`, se a mensagem tem mídia, chamar `DownloadMediaByPath` → `UploadAndPresign` → setar `msg.MediaURL`
- [ ] 1.5 Adicionar proteção contra re-download: pular se `msg.MediaURL` já está preenchido
- [ ] 1.6 Adicionar tratamento de erro: logar Warn em caso de falha no download, continuar persistência sem mídia

## 2. Query de Mensagens Não Importadas

- [ ] 2.1 Adicionar método `FindUnimportedHistory(ctx, sessionID, since, limit, offset) ([]Message, error)` ao `MessageRepo` interface e `MessageRepository` em `internal/repo/message.go`
- [ ] 2.2 Adicionar método `MarkImportedToChatwoot(ctx, sessionID, msgID string) error` ao `MessageRepo` interface e `MessageRepository`
- [ ] 2.3 Escrever testes unitários para `FindUnimportedHistory` e `MarkImportedToChatwoot` em `internal/repo/message_test.go`

## 3. Implementação do importHistory

- [ ] 3.1 Implementar lógica principal de `importHistory` em `internal/integrations/chatwoot/service.go`: buscar mensagens via `FindUnimportedHistory`, iterar com rate limiter (10/s), criar conversas e mensagens no Chatwoot, marcar como importadas
- [ ] 3.2 Implementar import de mensagens de texto: `findOrCreateConversation` → `client.CreateMessage` com `sourceID` → `MarkImportedToChatwoot`
- [ ] 3.3 Implementar import de mensagens com mídia: download do MinIO via presigned URL → `client.CreateMessageWithAttachment` → `MarkImportedToChatwoot`
- [ ] 3.4 Adicionar `singleflight.Group` ao `Service` para proteção contra import concorrente por sessão
- [ ] 3.5 Atualizar métrica `cw_history_import_progress` durante o processamento
- [ ] 3.6 Implementar `handleHistorySync` real: trigger automático de import se `cfg.ImportOnConnect=true` (reutilizar lógica do `handleConnected`)
- [ ] 3.7 Escrever testes unitários para `importHistory` em `internal/integrations/chatwoot/service_test.go`

## 4. Endpoint de Importação Manual

- [ ] 4.1 Implementar `Handler.ImportHistory` em `internal/integrations/chatwoot/handler.go`: chamar `Service.importHistory` em goroutine, retornar `202 Accepted` com status
- [ ] 4.2 Adicionar resposta de erro `404` quando Chatwoot não está configurado para a sessão
- [ ] 4.3 Escrever testes do handler em `internal/integrations/chatwoot/handler_test.go`

## 5. Hierarquia de Nomes de Contato

- [ ] 5.1 Criar interface `ContactNameGetter` com método `GetContactName(ctx, sessionID, jid) string` e implementar no `wa.Manager` consultando `Store.Contacts`
- [ ] 5.2 Injetar `ContactNameGetter` no `Service` via setter `SetContactNameGetter`
- [ ] 5.3 Atualizar `upsertConversation` em `conversation.go`: usar `ContactNameGetter.GetContactName` antes de `pushName` ao criar contato
- [ ] 5.4 Atualizar `handlePushName` em `inbound_events.go`: verificar se contato já tem nome não-numérico antes de sobrescrever
- [ ] 5.5 Escrever testes para hierarquia de nomes em `internal/integrations/chatwoot/contact_name_test.go`

## 6. Integração e Validação Final

- [ ] 6.1 Atualizar injeção de dependências em `cmd/wzap/main.go`: passar `wa.Manager` e `storage.Minio` ao `HistoryService`
- [ ] 6.2 Atualizar injeção de `ContactNameGetter` no Chatwoot `Service`
- [ ] 6.3 Executar `go mod tidy` e verificar compilação (`make build`)
- [ ] 6.4 Executar todos os testes (`go test -v -race ./...`)
- [ ] 6.5 Executar lint (`golangci-lint run ./...`)
