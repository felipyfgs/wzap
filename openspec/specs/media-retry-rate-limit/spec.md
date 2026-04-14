## ADDED Requirements

### Requirement: Rate limit nos media retry receipts do history sync

O sistema SHALL limitar a taxa de envio de `SendMediaRetryReceipt` durante o processamento de history sync a no máximo 1 request a cada 3 segundos por sessão.

O rate limiter MUST ser aplicado somente ao path de media retry do history sync — mensagens live e outros fluxos não são afetados.

O intervalo de rate limit MUST ser definido como constante no `HistoryService` (não configurável via env/API nesta versão).

#### Scenario: Media retry respeitando rate limit

- **WHEN** o history sync contém 10 mensagens com mídia expirada (erro 403/404/410)
- **THEN** o sistema envia no máximo 1 `SendMediaRetryReceipt` a cada 3 segundos
- **THEN** todas as 10 mídias são eventualmente processadas (~30 segundos)

#### Scenario: Mensagens sem mídia não sofrem delay

- **WHEN** o history sync contém 100 mensagens de texto
- **THEN** todas são persistidas imediatamente, sem nenhum delay de rate limit

#### Scenario: Mídia disponível para download direto não sofre delay

- **WHEN** o history sync contém mídia cujo download direto funciona (sem erro 403/404/410)
- **THEN** o download e upload para S3 ocorre sem nenhum delay de rate limit

### Requirement: Ticker é criado e parado corretamente

O `time.Ticker` MUST ser criado no início do processamento de cada history sync event e parado (`.Stop()`) ao final, evitando leak de goroutines.

#### Scenario: Cleanup do ticker

- **WHEN** o processamento do history sync event finaliza
- **THEN** o ticker é parado via `defer ticker.Stop()`
- **THEN** nenhum goroutine leak ocorre

### Requirement: Logging do rate limit

O sistema MUST logar em nível DEBUG quando um media retry é atrasado pelo rate limiter, incluindo session ID e message ID.

#### Scenario: Log de throttle

- **WHEN** um media retry é atrasado pelo rate limiter
- **THEN** o sistema loga: `"History sync media retry: aguardando rate limit"` com campos `session` e `mid`
