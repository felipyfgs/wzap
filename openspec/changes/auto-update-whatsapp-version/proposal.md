## Why

O WhatsApp recusou sessões do wzap com erro HTTP 405 porque o `whatsmeow` anunciava a versão expirada `2.3000.1035920091`. Como essas versões mudam com frequência, depender apenas da versão compilada exige rebuild e redeploy emergenciais para restabelecer as conexões.

## What Changes

- Resolver a versão do cliente WhatsApp uma vez durante a inicialização, antes de criar ou reconectar clientes.
- Consultar primeiro o `web.whatsapp.com` pelo mecanismo nativo do `whatsmeow` e usar o catálogo JSON do WPPConnect como fallback.
- Continuar a inicialização com a versão embutida quando as fontes automáticas estiverem indisponíveis e impedir downgrade automático.
- Adicionar `WA_VERSION` como override explícito, com precedência sobre consultas externas e validação de versões `x.y.z`, `x.y.z-alpha` e `x.y.z-beta`.
- Expor `WA_VERSION` nos ambientes Docker Compose de desenvolvimento e produção.
- Atualizar o `whatsmeow` para uma revisão que aplica `SetWAVersion` ao payload de login e adaptar a compatibilidade da API de download de mídia.

## Não-objetivos

- Não atualizar a versão periodicamente nem reiniciar sessões durante a execução.
- Não alterar endpoints HTTP, banco de dados, webhooks, eventos NATS, métricas ou frontend.
- Não modificar o change ou o pacote de integração do Chatwoot.

## Capabilities

### New Capabilities

- `whatsapp-client-version`: resolução automática, override operacional, fallback e aplicação segura da versão anunciada pelo cliente WhatsApp.

### Modified Capabilities

Nenhuma.

## Impact

- **Código**: `internal/config/` e `internal/wa/`.
- **Infraestrutura**: `.env.example`, `docker-compose.dev.yml` e `docker-compose.prod.yml`.
- **Dependências**: atualização do `go.mau.fi/whatsmeow` e dependências transitivas.
- **Operação**: até dez segundos adicionais na inicialização quando as fontes externas estiverem lentas; logs passam a informar versão e fonte escolhidas.

## Riscos e mitigações

- **Mudança no HTML do WhatsApp**: usar WPPConnect como fallback e manter a versão embutida se ambas as fontes falharem.
- **Versão externa incompatível com o protocolo compilado**: atualizar o `whatsmeow`, impedir downgrade automático e permitir pin manual por `WA_VERSION`.
- **Configuração manual inválida**: falhar cedo com erro descritivo antes de reconectar sessões.
