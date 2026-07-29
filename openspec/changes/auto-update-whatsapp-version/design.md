## Context

O `whatsmeow` embutia `2.3000.1035920091`, versão recusada pelo WhatsApp com HTTP 405. A biblioteca já oferece `GetLatestVersion`, que extrai `client_revision` do `web.whatsapp.com`, e `store.SetWAVersion`, mas a revisão usada pelo projeto não atualiza o `AppVersion` do payload de login. O WPPConnect mantém o mesmo revisionamento em um JSON público e serve como fonte secundária.

A versão é estado global do pacote `store`; portanto, deve ser definida uma vez, antes da criação ou reconexão de qualquer cliente. Não há fluxo handler → service → repo, mudança de banco, endpoint, métrica ou webhook nesta feature.

## Goals / Non-Goals

**Goals:**

- Evitar novas indisponibilidades causadas apenas por versão web expirada.
- Permitir pin operacional sem rebuild por `WA_VERSION`.
- Preservar disponibilidade quando as fontes externas falharem.
- Tornar a decisão observável por logs estruturados.

**Non-Goals:**

- Atualizar versões em background ou reconectar sessões automaticamente.
- Garantir compatibilidade com mudanças de protocolo que exijam novos protobufs.
- Alterar APIs, persistência, Chatwoot ou frontend.

## Decisions

### Resolver antes do sqlstore e dos clientes

`NewManager` executará a configuração de versão antes de `sqlstore.New`. Assim, `ReconnectAll` e `Connect` sempre clonam um `BaseClientPayload` já atualizado.

```text
server.SetupRoutes
       |
       v
wa.NewManager --> WA_VERSION preenchida? --sim--> validar/aplicar override
       | não
       v
web.whatsapp.com --falha--> WPPConnect --falha--> manter versão embutida
       |
       v
store.SetWAVersion --> sqlstore.New --> ReconnectAll
```

Alternativa descartada: atualizar periodicamente. A alteração não afeta conexões existentes e exigir reconexão automática ampliaria risco e escopo.

### Fonte oficial com fallback WPPConnect

O resolver usará `whatsmeow.GetLatestVersion` como fonte primária. Se ela falhar, fará GET do `versions.json` do WPPConnect e normalizará `currentVersion`, removendo apenas `-alpha` ou `-beta`.

Cada requisição terá timeout de cinco segundos sob um contexto total de dez segundos. Se ambas falharem, o erro será absorvido no modo automático e a versão embutida será mantida. Respostas do fallback serão lidas com limite de tamanho.

Alternativa descartada: depender somente do WPPConnect, pois é uma fonte de terceiros; ele é mais adequado como redundância.

### Override estrito e proteção contra downgrade

`WA_VERSION` não vazia elimina qualquer chamada externa. Cada componente será convertido com `strconv.ParseUint(..., 32)`; formatos inválidos e `0.0.0` falham cedo. Override manual pode fazer downgrade intencional. Resultado automático inferior ao embutido é rejeitado.

### Atualização do whatsmeow

O módulo será atualizado para `v0.0.0-20260722203353-e9a033b24933`, que corrige `SetWAVersion` e atualiza protobufs. A nova assinatura de `DownloadMediaWithPath` não recebe `fileLength` e exige o indicador de hash: o contrato interno manterá `fileLength` para não alterar consumidores, mas o adapter deixará de repassá-lo e exigirá hash para mídia criptografada.

## Risks / Trade-offs

- **HTML do WhatsApp mudar** → fallback WPPConnect e versão embutida.
- **As duas fontes demorarem** → cinco segundos por requisição e dez segundos totais.
- **Versão recente exigir protocolo novo** → dependência atualizada e override manual para rollback operacional.
- **Atualização transitiva quebrar APIs** → adaptar a assinatura de mídia e executar build, vet, lint e testes completos.
- **Log excessivo em falhas HTTP** → não incluir corpos de resposta; registrar apenas erro contextual e versão mantida.

## Migration Plan

1. Atualizar dependências e compatibilidade de compilação.
2. Implantar com `WA_VERSION` vazia e confirmar o log de versão antes das reconexões.
3. Se a resolução automática causar problema, fixar uma versão conhecida em `WA_VERSION` e reiniciar a API.
4. Para rollback completo, reverter código e dependências; não há estado persistente nem migração de banco.

## Open Questions

Nenhuma. Frequência, fontes, precedência e política de falha foram definidas antes da implementação.
