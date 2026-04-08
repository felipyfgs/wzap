## 1. Schema e modelo canônico

- [x] 1.1 Criar migration para a tabela `wz_chats` com chave por sessão e chat, campos de metadados do chat e `raw` para dados históricos relevantes
- [x] 1.2 Criar migration para enriquecer `wz_messages` com campos de origem e rastreabilidade da ingestão histórica
- [x] 1.3 Adicionar model de chat canônico em `internal/model/` e definir os campos usados pelo domínio
- [x] 1.4 Implementar repositório de chats em `internal/repo/` com operações de upsert e leitura por sessão/chat

## 2. Ingestão do HistorySync no wa.Manager

- [x] 2.1 Adicionar no `wa.Manager` um callback explícito para persistência de `*events.HistorySync`
- [x] 2.2 Atualizar `internal/wa/events.go` para chamar a persistência completa do `HistorySync` sem alterar o payload resumido atual de webhook/NATS
- [x] 2.3 Extrair do `HistorySync` os metadados de conversa necessários para `wz_chats`
- [x] 2.4 Extrair de `Conversation.Messages` as mensagens históricas e encaminhá-las para persistência canônica idempotente

## 3. Consolidação da persistência canônica

- [x] 3.1 Evoluir `internal/service/history.go` para suportar persistência de chats, mensagens live e chunks históricos
- [x] 3.2 Reutilizar `wz_messages` com upsert idempotente para live e history sync, preservando os melhores dados em conflitos
- [x] 3.3 Registrar aliases e metadados úteis para reconciliação entre `PN` e `LID`
- [x] 3.4 Integrar o novo repositório de chats na inicialização do servidor e no fluxo de persistência já ligado ao `wa.Manager`

## 4. Leitura futura e compatibilidade

- [x] 4.1 Garantir que o endpoint atual de histórico continue funcional após o enriquecimento de `wz_messages`
- [x] 4.2 Adicionar consultas mínimas para listar chats canônicos por sessão, preparando consumo futuro pelo importador do Chatwoot
- [x] 4.3 Registrar metadados suficientes para permitir ordenação posterior por timestamp e `chunkOrder`

## 5. Testes e validação

- [x] 5.1 Adicionar testes unitários para o repositório de chats e para o upsert enriquecido de mensagens
- [x] 5.2 Adicionar testes para a ingestão de `HistorySync`, cobrindo múltiplas conversas e chunks fora de ordem
- [x] 5.3 Adicionar testes para deduplicação entre mensagem live e mensagem histórica com mesmo identificador
- [x] 5.4 Executar `go test -v -race ./internal/...` cobrindo os pacotes afetados
- [x] 5.5 Executar `golangci-lint run ./...` e corrigir eventuais problemas
