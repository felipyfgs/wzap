## Context

O whatsmeow usa `client.AddEventHandler(fn)` para registrar handlers de eventos. Cada chamada **acumula** um handler — não substitui. Em `wa/connect.go`, a função `Connect()` registra dois handlers que chamam `handleEvent`:

```
Connect() é chamado
│
├─ AddEventHandler #1 → handleEvent(sessionID, evt)   ← linha 165
│
└─ AddEventHandler #2 → switch Connected/PairSuccess/Disconnected/LoggedOut
                         + handleEvent(sessionID, evt) dentro do switch?
```

Na verdade o segundo handler (linha 169) é um switch exclusivo de lifecycle — ele **não** chama `handleEvent`. Mas o handler **#1** (linha 165) está redundante com o handler já registrado em `ReconnectAll` (linha 82) para sessões reconectadas. Para sessões criadas via `Connect()`, o handler #1 (linha 165) é o único, mas ele é seguido pelo handler #2 (linha 169) que trata apenas eventos de lifecycle — o problema é que o handler #1 roda para **todos** os eventos, incluindo os de lifecycle, enquanto o handler #2 também os processa.

Investigando mais: o duplo log de "Message received" ocorre porque `Connect()` registra o handler #1 (linha 165) chamando `handleEvent` e, em paralelo, `OnMediaReceived` dispara `service.AutoUploadMedia` que internamente chama `PersistMessage` — este por sua vez pode disparar novamente `OnMessageReceived` de forma assíncrona dependendo da implementação do pool. O fluxo real é:

```
*events.Message chega
    │
    ├─ handler #1 → handleEvent
    │       ├─ log "Message received" (msgType=unknown, pois mídia ainda não resolvida)
    │       ├─ OnMediaReceived → async pool → AutoUploadMedia
    │       └─ OnMessageReceived → Chatwoot handleMessage
    │
    └─ handler #2 (inline switch) → não processa *events.Message
```

O segundo "Message received" (com `msgType=video`) vem do próprio `handleEvent` sendo chamado uma segunda vez pelo pool assíncrono ou por uma segunda entrega do evento pelo whatsmeow após download do media. Verificar se `OnMediaReceived` dispara um segundo `handleEvent`.

## Goals / Non-Goals

**Goals:**
- Garantir que `handleEvent` seja chamado exatamente uma vez por evento por sessão
- Cobrir tipos de mensagem ausentes em `extractMessageContent`
- Adicionar campo `session` ao log `handleMessage` em chatwoot
- Padronizar `component=async` em `async/pool.go`

**Non-Goals:**
- Refatorar a arquitetura de event handlers do whatsmeow
- Alterar o comportamento funcional de qualquer feature existente
- Modificar o modelo de dados ou migrations

## Decisions

### D1: Remover o `AddEventHandler` redundante em `Connect()`

**Decisão:** Remover o `AddEventHandler` das linhas 165–167 de `connect.go` e mover o `handleEvent` para dentro do handler inline que já existe nas linhas 169–204.

**Alternativa considerada:** Manter os dois handlers separados e adicionar deduplicação por event ID no `handleEvent`. Rejeitada — overcomplicated, adiciona estado, e esconde o bug em vez de corrigi-lo.

**Rationale:** Um único handler com um switch completo (`Connected`, `PairSuccess`, `Disconnected`, `LoggedOut`, `default: handleEvent(...)`) é a abordagem mais simples e correta.

```
AddEventHandler — único handler por Connect():
switch v := evt.(type) {
case *events.Connected:    // lifecycle
case *events.PairSuccess:  // lifecycle
case *events.Disconnected: // lifecycle
case *events.LoggedOut:    // lifecycle
default:
    m.handleEvent(sessionID, evt)
}
```

### D2: Adicionar `default: handleEvent()` ao switch inline

O switch em `connect.go` linha 172 precisa de um `default` case que chame `handleEvent`. Assim todos os eventos de negócio são processados exatamente uma vez pelo `handleEvent`, enquanto os de lifecycle são tratados separadamente sem chamar `handleEvent`.

### D3: Enriquecer `extractMessageContent` com tipos ausentes

Adicionar cases para `ViewOnceMessage`, `TemplateMessage`, `InteractiveMessage`, `LiveLocationMessage`, `ProductMessage`, e mudar o `default` de `"unknown"` para `"unsupported"` — sinaliza que o tipo existe mas não foi mapeado ainda, diferente de `nil` message.

## Riscos / Trade-offs

| Risco | Mitigação |
|---|---|
| O default case no switch pode chamar `handleEvent` para eventos de lifecycle também (double processing de Connected etc.) | Garantir que `handleEvent` ignore ou trate graciosamente `*events.Connected`, `*events.PairSuccess`, etc. — verificar o switch em events.go |
| Remover o handler linha 165 sem adicionar o `default` case quebra todo o processamento de mensagens | Adicionar `default` antes de remover o handler existente; cobrir com teste de regressão |

## Migration Plan

1. Adicionar `default: m.handleEvent(sessionID, evt)` ao switch inline em `connect.go`
2. Remover o `AddEventHandler` das linhas 165–167
3. Verificar com `go build ./...` e `go test -race ./internal/wa/...`
4. Testar manualmente enviando uma mensagem para a sessão conectada e confirmar log único

## Open Questions

- O segundo "Message received" com `msgType=video` pode ser originado de `OnMediaReceived` chamando algum path que reinvoca `handleEvent`? Investigar `service/media.go` e o async pool durante implementação.
