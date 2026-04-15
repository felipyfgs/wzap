## Context

O whatsmeow despacha mensagens de status como `events.Message` com `Chat.Server == BroadcastServer`. Hoje, `wa.Manager` roteia tudo via `OnMessageReceived` para `HistoryService.PersistMessage`, que salva em `wz_messages`. O `FindBySession` do `MessageRepo` retorna status misturados com mensagens reais.

O envio de status funciona via `client.SendMessage(ctx, types.StatusBroadcastJID, msg)` no `service/message_status.go`, usando `persistSent` que salva em `wz_messages`.

Frontend: status forms ja existem em `components/sessions/message-forms/` mas estao dentro do `SendMessageModal`, misturados com mensagens normais. Nao existe pagina dedicada para visualizar status recebidos.

## Goals / Non-Goals

**Goals:**
- Separar armazenamento de status (stories) de mensagens normais
- Criar API REST dedicada para envio e listagem de status
- Implementar `IgnoreStatus` no roteamento de eventos whatsmeow
- Criar pagina frontend com visual de stories para visualizar e enviar status
- Migrar dados existentes de `wz_messages` para `wz_statuses`

**Non-Goals:**
- Baixar media de status recebidos automaticamente (pode ser feito sob demanda)
- Suporte a reply/forward de status
- Expiracao automatica de status (stories expiram em 24h no WhatsApp, mas o cleanup pode ser manual)
- Suporte a Cloud API para status (somente whatsmeow)

## Decisions

### 1. Tabela dedicada `wz_statuses` vs filtro em `wz_messages`

**Decisao**: Tabela dedicada `wz_statuses`.

**Alternativas consideradas**:
- (A) Filtro `WHERE chat_jid NOT LIKE 'status@%'` em todas as queries de mensagem — simples mas frágil, polui a tabela com dados irrelevantes
- (B) Partial index `WHERE chat_jid NOT LIKE 'status@%'` — melhora performance mas nao resolve a poluicao de dados
- (C) Tabela dedicada — separacao limpa, queries simples, indexes otimizados para o padrao de acesso de status

**Racional**: Status tem padrao de acesso completamente diferente (agrupado por contato, timeframe curto, leitura sequencial). Tabela dedicada permite indexes e queries otimizados.

### 2. Roteamento de eventos no `wa/events.go`

**Decisao**: Verificar `Chat.Server == types.BroadcastServer` no handler de `events.Message` e rotear para callback `OnStatusReceived`.

**Detalhe**: O whatsmeow nao tem evento dedicado para status. Mensagens de status de outros contatos chegam como `events.Message` com `Chat` no servidor `broadcast`. O campo `Chat.User == "status"` indica `StatusBroadcastJID`. Status de outros contatos tem `Chat` diferente (broadcast list JID) mas ainda com servidor `broadcast`.

```
events.Message recebido
       │
       ▼
┌──────────────────────────────┐
│ Chat.Server == BroadcastServer? │
└──────────┬───────────────────┘
     Yes   │   No
           ▼      ▼
   ┌──────────┐  ┌──────────────┐
   │ OnStatus │  │ OnMessage    │
   │ Received │  │ Received     │
   └────┬─────┘  └──────────────┘
        │
        ▼
   ┌──────────────┐
   │ IgnoreStatus │──Yes──▶ Discard
   │  setting?    │
   └──────┬───────┘
          │ No
          ▼
   Persist wz_statuses
   Dispatch event
```

### 3. Callback `OnStatusReceived` no `wa.Manager`

**Decisao**: Adicionar callback `OnStatusReceived(sessionID, msgID, chatJID, senderJID, fromMe, msgType, body, mediaType, timestamp, raw)` seguindo o mesmo padrao de `OnMessageReceived`.

### 4. Implementacao de `IgnoreStatus`

**Decisao**: Checar o setting `SessionSettings.IgnoreStatus` no `wa/events.go` antes de despachar. O `Manager` precisa acessar o repo de sessao para ler o setting, ou receber um callback/resolver externamente.

**Alternativas**:
- (A) Passar `SessionSettings` para o `handleEvent` — adiciona acoplamento
- (B) Callback `ShouldIgnoreStatus(sessionID) bool` no Manager — flexivel, injetado pelo `server.go`
- (C) Filtro no service/handler level — status ja persistido, desperdicio

**Racional**: Opcao B — callback injetado no Manager, similar ao padrao de outros callbacks. O `server.go` configura com a logica de leitura do repo.

### 5. Historico de status recebidos de outros contatos

**Decisao**: Persistir status recebidos de outros contatos na tabela `wz_statuses`. O `sender_jid` sera o JID do contato que publicou o status.

### 6. Frontend: pagina dedicada vs tab dentro de Messages

**Decisao**: Pagina dedicada `/sessions/:id/status` com visual de stories.

**Racional**: Status tem UX completamente diferente de mensagens (visualizacao fullscreen, navegacao entre stories, agrupamento por contato). Tab misturado poluiria a pagina de mensagens.

## Risks / Trade-offs

- **[Status de outros contatos com Chat JID ambiguo]** → O whatsmeow pode enviar status recebidos com diferentes formatos de JID. Precisamos testar e cobrir multiplos padroes (`status@broadcast`, broadcast lists). Mitigacao: logar JIDs nao reconhecidos para ajuste.
- **[Breaking change em GET /messages]** → Clientes que dependem de status nas listas de mensagem vao perder esses dados. Mitigacao: migration copia dados antes de aplicar filtros; documentar na changelog.
- **[Chatwoot]** → Ja ignora `status@broadcast` mas pode precisar filtro generico para `status@`. Mitigacao: adicionar `shouldIgnoreJID` generico no `inbound_message.go`.
- **[Nao implementacao de expiracao]** → Status acumulados indefinidamente. Mitigacao: endpoint `DELETE /status/:senderJid` para limpeza manual; pode adicionar TTL batch job depois.
