## ADDED Requirements

### Requirement: Preservação da superfície pública externa

O pacote `internal/integrations/chatwoot/` DEVE manter exatamente a mesma superfície pública consumida por callers externos (`internal/server/router.go`, `internal/handler/session.go`, `internal/handler/cloud_api.go`). Nenhum símbolo exportado atualmente importado fora do pacote PODE ser removido, renomeado ou ter sua assinatura alterada por este refactor.

#### Cenário: Build do módulo permanece verde após refactor

- **WHEN** o comando `go build ./...` é executado após qualquer passo do refactor
- **THEN** o build conclui com exit code 0 e nenhum erro de símbolo indefinido

#### Cenário: Consumidores externos continuam compilando sem alteração

- **WHEN** arquivos em `internal/server/router.go`, `internal/handler/session.go` e `internal/handler/cloud_api.go` não recebem modificação
- **THEN** esses arquivos continuam compilando e passando em `go vet ./...`

#### Cenário: Testes existentes passam sem mudança lógica

- **WHEN** `go test ./internal/integrations/chatwoot/...` é executado após cada passo
- **THEN** 100% dos testes existentes passam sem modificação na lógica de asserções; apenas nomes de arquivo de teste podem mudar (quando o arquivo testado for renomeado)

### Requirement: Ausência de código morto confirmado

O pacote `internal/integrations/chatwoot/` NÃO DEVE conter funções exportadas, arquivos ou blocos de código que este refactor identificou como mortos (`buildCloudReactionMessage`, wrapper `UnlockCloudWindow` sobre `unlockCloudWindow`, arquivo `qrcode.go`, anchor `var _ model.EventType = ""` em `inbox.go`, helper duplicado `urlFilename`).

#### Cenário: buildCloudReactionMessage removido

- **WHEN** `grep -rn "buildCloudReactionMessage" internal/integrations/chatwoot/` é executado
- **THEN** retorna zero ocorrências

#### Cenário: qrcode.go deletado

- **WHEN** `ls internal/integrations/chatwoot/qrcode.go` é executado
- **THEN** retorna erro "No such file or directory"

#### Cenário: Apenas uma implementação de extração de nome de arquivo por URL

- **WHEN** `grep -rn "func.*ilename.*URL\|func urlFilename" internal/integrations/chatwoot/` é executado
- **THEN** exatamente uma função é definida (`filenameFromURL`)

### Requirement: Limite de tamanho de arquivo

Após o refactor, nenhum arquivo `.go` não-teste em `internal/integrations/chatwoot/` DEVE exceder 450 LOC. Arquivos que hoje excedem esse limite (`wa_events.go` 663, `cw_webhook.go` 507, `inbox_cloud.go` 590, `parser.go` 591) DEVEM ser divididos em unidades coesas menores.

#### Cenário: Nenhum arquivo grande restante

- **WHEN** `find internal/integrations/chatwoot -name '*.go' -not -name '*_test.go' -exec wc -l {} \;` é executado após o refactor completo
- **THEN** nenhum arquivo reporta mais de 450 linhas

#### Cenário: service.go reduzido

- **WHEN** `wc -l internal/integrations/chatwoot/service.go` é executado após o refactor
- **THEN** `service.go` reporta entre 150 e 220 linhas (redução de ~433 → ~180 via extração de `import.go` e `events.go`)

### Requirement: Padronização de nomenclatura de arquivos

Arquivos `.go` em `internal/integrations/chatwoot/` NÃO DEVEM usar prefixo `cw_` ou `wa_` quando o contexto do pacote já torna o significado claro. Prefixos descritivos (`inbox_`, `cloud_`, `webhook_`, `events_`, `message_`) PODEM ser usados quando desambiguam subdomínio.

#### Cenário: Prefixos cw_ e wa_ removidos

- **WHEN** `ls internal/integrations/chatwoot/cw_*.go internal/integrations/chatwoot/wa_*.go 2>/dev/null` é executado após o refactor
- **THEN** retorna lista vazia

#### Cenário: Histórico git preservado

- **WHEN** `git log --follow internal/integrations/chatwoot/conversation.go` é executado
- **THEN** o log mostra commits anteriores do arquivo `cw_conversation.go` (rename via `git mv` preserva histórico)

### Requirement: Eliminação de duplicação de prólogo entre inbox handlers

A lógica comum de entrada de mensagens (parse do payload → resolução LID → filtro de JID ignorado → verificação de idempotência) NÃO DEVE estar duplicada entre `inbox_api.go` e `inbox_cloud.go`. DEVE existir um helper compartilhado em `inbox_common.go` consumido por ambos os handlers.

#### Cenário: Helper inboxPrologue existe e é chamado pelos dois handlers

- **WHEN** `grep -n "inboxPrologue" internal/integrations/chatwoot/` é executado após o refactor
- **THEN** a função é definida em `inbox_common.go` e chamada em `inbox_api.go` e `inbox_cloud.go`

#### Cenário: Paridade comportamental entre modos API e Cloud

- **WHEN** o mesmo payload de mensagem é processado pelo handler API e pelo handler Cloud (com flags apropriadas para idempotência de DB)
- **THEN** a decisão de "processar vs. pular" é equivalente em ambos os modos para as etapas compartilhadas (parse, LID, filtro, idempotência em cache)

### Requirement: Pacote permanece flat

O pacote `internal/integrations/chatwoot/` NÃO DEVE introduzir subdiretórios como parte deste refactor. Toda a organização DEVE ser feita via nomes de arquivo dentro do mesmo pacote Go.

#### Cenário: Sem subdiretórios criados

- **WHEN** `find internal/integrations/chatwoot -mindepth 1 -type d` é executado após o refactor
- **THEN** retorna lista vazia
