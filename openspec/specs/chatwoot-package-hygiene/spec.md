## ADDED Requirements

### Requirement: Preservação da superfície pública externa

O pacote `internal/integrations/chatwoot/` MUST manter (DEVE manter) exatamente a mesma superfície pública consumida por callers externos (`internal/server/router.go`, `internal/handler/session.go`). Nenhum símbolo exportado atualmente importado fora do pacote PODE ser removido, renomeado ou ter sua assinatura alterada sem justificativa explícita em change proposal.

#### Cenário: Build do módulo permanece verde após refactor

- **WHEN** o comando `go build ./...` é executado após qualquer passo do refactor
- **THEN** o build conclui com exit code 0 e nenhum erro de símbolo indefinido

#### Cenário: Consumidores externos continuam compilando sem alteração

- **WHEN** arquivos em `internal/server/router.go` e `internal/handler/session.go` não recebem modificação
- **THEN** esses arquivos continuam compilando e passando em `go vet ./...`

#### Cenário: Testes existentes passam sem mudança lógica

- **WHEN** `go test ./internal/integrations/chatwoot/...` é executado após cada passo
- **THEN** 100% dos testes restantes passam sem modificação na lógica de asserções; testes exclusivos do modo Cloud (`inbox_cloud_test.go`, `mapping_test.go`, `cloud_api` fixtures) são removidos junto com a feature

### Requirement: Ausência de código morto confirmado

O pacote `internal/integrations/chatwoot/` MUST NOT (NÃO DEVE) conter funções exportadas, arquivos ou blocos de código identificados como mortos. Após a remoção do modo Cloud, verificações específicas incluem: ausência de `buildCloudReactionMessage`, `UnlockCloudWindow`, `buildCloudTextMessage`, `buildCloudMediaMessage`, `buildCloudLocationMessage`, `buildCloudContactMessage`, `buildCloudWebhookEnvelope`, `buildCloudContact`, `postToChatwootCloud`, `uploadCloudMedia`, `cloudMediaType`, `resolveCloudRefAsync`, `resolveCloudRefViaAPI`, `resolveAndPersistMessageRef`, `ResolveMessageBySourceID`, `ResolveConversationForContactPhone`, `BackfillCloudRefs`.

#### Cenário: Símbolos do modo Cloud removidos

- **WHEN** `grep -rn "buildCloud\|postToChatwootCloud\|uploadCloudMedia\|resolveCloudRef\|BackfillCloudRefs\|ResolveMessageBySourceID" internal/integrations/chatwoot/` é executado
- **THEN** retorna zero ocorrências

#### Cenário: Arquivos Cloud-only deletados

- **WHEN** `ls internal/integrations/chatwoot/inbox_cloud.go internal/integrations/chatwoot/mapping.go internal/integrations/chatwoot/mapping_test.go internal/integrations/chatwoot/inbox_cloud_test.go 2>/dev/null` é executado
- **THEN** retorna lista vazia (todos os arquivos foram removidos)

#### Cenário: cloud_api handler e DTO removidos

- **WHEN** `ls internal/handler/cloud_api.go internal/handler/cloud_api_test.go internal/dto/cloud_api.go 2>/dev/null` é executado
- **THEN** retorna lista vazia

#### Cenário: Apenas uma implementação de extração de nome de arquivo por URL

- **WHEN** `grep -rn "func.*ilename.*URL\|func urlFilename" internal/integrations/chatwoot/` é executado
- **THEN** exatamente uma função é definida (`filenameFromURL`)

### Requirement: Limite de tamanho de arquivo

Após o refactor, nenhum arquivo `.go` não-teste em `internal/integrations/chatwoot/` MUST (DEVE) exceder 450 LOC. Com a remoção do modo Cloud, `inbox_cloud.go` e `mapping.go` deixam de existir e a meta continua aplicável aos arquivos restantes.

#### Cenário: Nenhum arquivo grande restante

- **WHEN** `find internal/integrations/chatwoot -name '*.go' -not -name '*_test.go' -exec wc -l {} \;` é executado após o refactor completo
- **THEN** nenhum arquivo reporta mais de 450 linhas

#### Cenário: service.go reduzido

- **WHEN** `wc -l internal/integrations/chatwoot/service.go` é executado após o refactor
- **THEN** `service.go` reporta entre 100 e 200 linhas (depois da remoção de `processOutbound` cross-mode, referências a Cloud e setters de dependências Cloud-only)

### Requirement: Padronização de nomenclatura de arquivos

Arquivos `.go` em `internal/integrations/chatwoot/` NÃO DEVEM usar prefixo `cw_` ou `wa_` quando o contexto do pacote já torna o significado claro. Prefixos descritivos (`inbox_`, `cloud_`, `webhook_`, `events_`, `message_`) PODEM ser usados quando desambiguam subdomínio.

#### Cenário: Prefixos cw_ e wa_ removidos

- **WHEN** `ls internal/integrations/chatwoot/cw_*.go internal/integrations/chatwoot/wa_*.go 2>/dev/null` é executado após o refactor
- **THEN** retorna lista vazia

#### Cenário: Histórico git preservado

- **WHEN** `git log --follow internal/integrations/chatwoot/conversation.go` é executado
- **THEN** o log mostra commits anteriores do arquivo `cw_conversation.go` (rename via `git mv` preserva histórico)

### Requirement: Pacote permanece flat

O pacote `internal/integrations/chatwoot/` NÃO DEVE introduzir subdiretórios como parte deste refactor. Toda a organização DEVE ser feita via nomes de arquivo dentro do mesmo pacote Go.

#### Cenário: Sem subdiretórios criados

- **WHEN** `find internal/integrations/chatwoot -mindepth 1 -type d` é executado após o refactor
- **THEN** retorna lista vazia
