## Why

O whatsmeow entrega `*events.Message` **duas vezes** para a mesma mensagem quando `AutomaticMessageRerequestFromPhone = true` está ativo. Isso ocorre para mensagens cujo conteúdo não pode ser descriptografado imediatamente (especialmente de senders com JID `@lid` — linked device IDs): a primeira entrega carrega o envelope parcial (`msgType=unknown`), e após obter a chave do dispositivo remoto, whatsmeow entrega novamente com o conteúdo completo e o tipo correto. Como resultado, todo o pipeline de processamento executa em duplicidade: log de "Message received", webhook, e handler Chatwoot são disparados duas vezes para o mesmo `mid`.

## What Changes

- Implementar deduplicação de `*events.Message` em `internal/wa/events.go`: ignorar a primeira entrega parcial (`msgType=unknown`) e processar apenas a entrega completa
- Adicionar cobertura de tipos ausentes em `extractMessageContent` (`ViewOnceMessage`, `TemplateMessage`, `InteractiveMessage`, `LiveLocationMessage`, `ProductMessage`) e renomear o fallback de `"unknown"` para `"unsupported"`
- Adicionar campo `session` ao log `handleMessage` em `internal/integrations/chatwoot/inbound_message.go`
- Adicionar `component=async` nos logs de `internal/async/pool.go`

## Capabilities

### New Capabilities

Nenhuma nova capability.

### Modified Capabilities

Nenhuma mudança de requisitos de spec.

## Impact

- `internal/wa/events.go` — lógica de deduplicação para `*events.Message` com `msgType=unknown`; `extractMessageContent` enriquecida
- `internal/integrations/chatwoot/inbound_message.go` — campo `session` adicionado ao log `handleMessage`
- `internal/async/pool.go` — campo `component=async` adicionado

## Não-objetivos

- Não desabilitar `AutomaticMessageRerequestFromPhone` ou `UseRetryMessageStore` — são features necessárias para entrega confiável de mensagens
- Não reescrever o fluxo de gerenciamento de clientes whatsmeow
- Não alterar comportamento funcional do processamento de mensagens além da deduplicação
- Não modificar testes existentes além do necessário

## Riscos e mitigações

| Risco | Mitigação |
|---|---|
| Descartar a primeira entrega pode perder mensagens que chegam apenas uma vez com `msgType=unknown` | Garantir que tipos genuinamente não suportados retornem `"unsupported"` (não `"unknown"`); monitorar com log de nível WARN para drops |
| Cache de deduplicação cresce indefinidamente em sessões de alta carga | Usar TTL de 30s com eviction automática; limitar tamanho máximo do cache |
| `extractMessageContent` retornar tipo incorreto para mensagens complexas | Cobrir os casos mais comuns e manter `"unsupported"` como fallback explícito |
