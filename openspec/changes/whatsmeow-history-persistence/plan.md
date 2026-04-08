## Plano Simples

### Objetivo

Salvar histórico e conversas recebidos pelo `whatsmeow` no banco do WZAP para permitir importação futura e confiável no Chatwoot.

### Situação Atual

- O projeto já persiste mensagens em `wz_messages`
- O `whatsmeow` já emite `events.Message` e `events.HistorySync`
- O `HistorySync` hoje é resumido para webhook/NATS, mas seu conteúdo completo não é persistido no domínio do WZAP
- Ainda não existe uma tabela própria de conversas/chats do WZAP
- O importador de histórico do Chatwoot ainda está em modo placeholder

### Plano Proposto

#### Fase 1 — Base canônica de histórico

- Criar tabela `wz_chats` para armazenar metadados de conversa por sessão
- Expandir `wz_messages` para distinguir origem do dado (`live` vs `history_sync`)
- Criar repositório e model de chat próprios do domínio
- Manter `whatsmeow_*` apenas como infraestrutura interna da lib, não como fonte principal do produto

#### Fase 2 — Ingestão do `HistorySync`

- Adicionar um callback no `wa.Manager` para processar `*events.HistorySync`
- Persistir cada `Conversation` recebida no blob histórico em `wz_chats`
- Persistir cada `HistorySyncMsg.Message` em `wz_messages` com upsert idempotente
- Preservar o webhook/NATS atual apenas com payload resumido

#### Fase 3 — Consolidação de dados vivos

- Continuar persistindo mensagens do fluxo ao vivo no mesmo repositório canônico
- Atualizar `wz_chats` com último timestamp, nome, flags e estado do chat
- Normalizar JIDs quando houver diferença entre `PN` e `LID`

#### Fase 4 — Importação para Chatwoot

- Implementar leitura paginada do banco por sessão/chat/período
- Resolver ou criar conversas no Chatwoot a partir de `wz_chats`
- Reproduzir mensagens históricas em ordem cronológica
- Registrar checkpoint para permitir retomada segura em falhas

### Escopo Inicial Recomendado

- Persistir `HistorySync` completo no banco
- Criar `wz_chats`
- Enriquecer `wz_messages`
- Deixar a importação completa para uma segunda etapa

### Riscos Principais

- Duplicidade entre mensagens do histórico e mensagens ao vivo
- Variação de identidade entre `LID` e número de telefone
- Importações grandes no Chatwoot gerando rate limit
- `HistorySync` chegar em múltiplos chunks e fora de ordem

### Mitigações

- Usar upsert por `session_id + message_id`
- Guardar aliases de JID e aproveitar mapeamentos do `whatsmeow`
- Importar em lotes com checkpoint
- Ordenar reprocessamento por timestamp e registrar `chunkOrder`

### Próximos Passos

- Escrever uma OpenSpec formal para a persistência canônica
- Definir schema inicial de `wz_chats` e campos extras de `wz_messages`
- Identificar exatamente onde ligar o callback de `HistorySync`
- Implementar primeiro a persistência, depois o import para Chatwoot
