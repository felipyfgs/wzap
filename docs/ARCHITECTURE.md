# wzap — Arquitetura e Visão Geral

## O que é

wzap é uma API HTTP que expõe as funcionalidades do WhatsApp como serviço REST. Cada instância do WhatsApp é representada como uma **sessão** isolada. A API permite criar múltiplas sessões em paralelo, enviar/receber mensagens, gerenciar grupos, contatos, newsletters, comunidades e entregar eventos em tempo real via webhooks.

O motor de integração com WhatsApp é a biblioteca [whatsmeow](https://github.com/tulir/whatsmeow), que implementa o protocolo WhatsApp Web multi-device.

---

## Stack de infraestrutura

| Componente | Tecnologia | Papel |
|---|---|---|
| HTTP Framework | [Fiber v2](https://github.com/gofiber/fiber) | Roteamento e handlers |
| Banco de dados | PostgreSQL 16 + pgx v5 | Persistência de sessões e webhooks |
| Mensageria | NATS JetStream | Publicação de eventos e fila de entrega de webhooks |
| Armazenamento | MinIO (S3-compatível) | Upload de mídia (imagens, vídeos, documentos) |
| WA Engine | whatsmeow | Protocolo WhatsApp multi-device |
| Logger | zerolog | Logs estruturados em JSON |

---

## Estrutura de pacotes

```
cmd/wzap/
  main.go               Entrypoint: inicializa infraestrutura e inicia o servidor

internal/
  config/               Carrega variáveis de ambiente para Config struct
  database/             Pool pgxpool + execução de migrations embarcadas
  broker/               Conexão NATS JetStream + publish de eventos
  storage/              Cliente MinIO para upload de mídia
  model/                Tipos de domínio: Session, Webhook, EventType, ...
  dto/                  Payloads de request/response HTTP
  repo/                 Acesso ao PostgreSQL (SessionRepository, WebhookRepository)
  middleware/           Auth (token global + token de sessão), Logger, Recovery, RequiredSession
  wa/                   Engine whatsmeow: Manager de clientes, connect/disconnect, QR, eventos
  webhook/              Entrega de webhooks: NATS consumer + HTTP POST com retry e HMAC
  service/              Lógica de negócio (bridge entre handlers e wa.Manager)
  handler/              Controllers HTTP Fiber
  server/               Bootstrap do Fiber App + registro de rotas e DI
```

---

## Autenticação

Dois níveis de acesso, resolvidos pelo middleware `Auth` via header `ApiKey`:

```
ApiKey: <token>
```

### Admin
O token coincide com a variável de ambiente `API_KEY`. Tem acesso completo: pode criar e listar sessões de qualquer usuário.

### Sessão
O token é o `apiKey` de uma sessão específica (gerado na criação). Só pode operar sobre essa sessão. Tentativas de acessar outra sessão retornam `403 Forbidden`.

### Sem autenticação configurada
Se `API_KEY` estiver vazio, todos os requests são tratados como admin.

**Fluxo de resolução:**
```
Header ApiKey
  ├── == API_KEY global  →  role = "admin"
  ├── == apiKey de sessão  →  role = "session", sessionId = session.ID
  └── inválido  →  401 Unauthorized
```

O middleware `RequiredSession` (aplicado às rotas `/sessions/:sessionId/*`) resolve o `:sessionId` por nome ou UUID e valida que um token de sessão não está acessando uma sessão alheia.

---

## Ciclo de vida de uma sessão

```
POST /sessions              Cria sessão no banco (status: disconnected)
  │
POST /sessions/:id/connect  Conecta ao WhatsApp
  │
  ├── Sessão já pareada (JID salvo)
  │     └── client.Connect() → status: connected
  │
  └── Sessão nova (sem JID)
        └── GetQRChannel() → client.Connect() → status: connecting
              │
              GET /sessions/:id/qr   Polling do QR code (salvo no banco)
              │
              [usuário escaneia QR]
              │
              events.PairSuccess → JID salvo, status: connected
              │
              events.Connected  → confirmação de conexão ativa

DELETE /sessions/:id        Remove sessão do banco e desconecta o cliente
POST /sessions/:id/disconnect  Desconecta sem remover
```

O `wa.Manager` mantém um mapa `sessionID → *whatsmeow.Client` protegido por `sync.RWMutex`. O `client.Connect()` é chamado **fora do lock** para não bloquear o `GetClient()` de outras sessões.

---

## Sistema de eventos e webhooks

### Publicação de eventos (NATS)

Todo evento recebido do whatsmeow (mensagem, receipt, conexão, desconexão, etc.) é serializado como JSON e publicado no subject `wzap.events.<sessionId>` do NATS JetStream (stream `WZAP_EVENTS`).

### Entrega de webhooks (HTTP)

Cada sessão pode ter múltiplos webhooks registrados. Cada webhook filtra por tipos de evento (`events` JSONB). Ao receber um evento:

```
wa/events.go: handleEvent()
  └── webhook.Dispatcher.Dispatch(sessionID, eventType, payload)
        │
        ├── webhook.NatsEnabled == true
        │     └── Publica em wzap.webhook.deliver.<webhookId> (NATS JetStream)
        │           └── StartConsumer() consome e entrega via HTTP com retry
        │
        └── webhook.NatsEnabled == false
              └── HTTP POST direto (goroutine)
```

**Entrega via NATS** usa o stream `WZAP_WEBHOOKS` com consumer `webhook-dispatcher`:
- Máximo 5 tentativas com backoff: 10s → 30s → 1m → 5m
- Assinatura HMAC-SHA256 no header `X-Wzap-Signature: sha256=<hash>` quando `secret` configurado
- Header `X-Wzap-Event: <tipo>` em todas as entregas

### Tipos de evento suportados

Mensagens, receipts, conexão/desconexão, pairing, grupos, presença, chamadas, labels, newsletters, sync, privacidade — mapeados como constantes `EventType` em `internal/model/events.go`. O tipo especial `All` subscreve todos os eventos.

---

## Rotas disponíveis

Base: `http://host:8080`

### Públicas (sem auth)
| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | Status da API e dependências |
| GET | `/swagger/*` | Swagger UI |

### Admin
| Método | Rota | Descrição |
|---|---|---|
| POST | `/sessions` | Criar sessão |
| GET | `/sessions` | Listar sessões |

### Sessão (prefixo `/sessions/:sessionId`)
| Método | Rota | Descrição |
|---|---|---|
| GET | `/` | Detalhes da sessão |
| DELETE | `/` | Remover sessão |
| POST | `/connect` | Conectar ao WhatsApp |
| POST | `/disconnect` | Desconectar |
| GET | `/qr` | QR code atual (PNG base64 + string) |
| POST | `/messages/text` | Enviar texto |
| POST | `/messages/image` | Enviar imagem |
| POST | `/messages/video` | Enviar vídeo |
| POST | `/messages/document` | Enviar documento |
| POST | `/messages/audio` | Enviar áudio |
| POST | `/messages/contact` | Enviar contato (vCard) |
| POST | `/messages/location` | Enviar localização |
| POST | `/messages/poll` | Criar enquete |
| POST | `/messages/sticker` | Enviar sticker |
| POST | `/messages/link` | Enviar link com preview |
| POST | `/messages/edit` | Editar mensagem enviada |
| POST | `/messages/delete` | Apagar mensagem |
| POST | `/messages/reaction` | Reagir a mensagem |
| POST | `/messages/read` | Marcar como lida |
| POST | `/messages/presence` | Definir presença (digitando, gravando) |
| GET | `/contacts` | Listar contatos |
| POST | `/contacts/check` | Verificar se números têm WhatsApp |
| POST | `/contacts/avatar` | Foto de perfil de um contato |
| POST | `/contacts/block` | Bloquear contato |
| POST | `/contacts/unblock` | Desbloquear contato |
| GET | `/contacts/blocklist` | Lista de bloqueados |
| POST | `/contacts/info` | Informações de JIDs |
| GET | `/contacts/privacy` | Configurações de privacidade |
| POST | `/contacts/profile-picture` | Atualizar foto de perfil |
| GET | `/groups` | Listar grupos |
| POST | `/groups/create` | Criar grupo |
| POST | `/groups/info` | Informações de grupo |
| POST | `/groups/invite-info` | Preview de grupo via link |
| POST | `/groups/join` | Entrar via link |
| POST | `/groups/invite-link` | Obter/resetar link de convite |
| POST | `/groups/leave` | Sair do grupo |
| POST | `/groups/participants` | Adicionar/remover/promover participantes |
| POST | `/groups/requests` | Listar solicitações de entrada |
| POST | `/groups/requests/action` | Aprovar/rejeitar solicitações |
| POST | `/groups/name` | Alterar nome |
| POST | `/groups/description` | Alterar descrição |
| POST | `/groups/photo` | Alterar foto |
| POST | `/groups/announce` | Modo somente admins |
| POST | `/groups/locked` | Bloquear edição de info |
| POST | `/groups/join-approval` | Ativar aprovação de entrada |
| POST | `/chat/archive` | Arquivar conversa |
| POST | `/chat/mute` | Silenciar conversa |
| POST | `/chat/pin` | Fixar conversa |
| POST | `/chat/unpin` | Desafixar conversa |
| POST | `/label/chat` | Adicionar label à conversa |
| POST | `/label/message` | Adicionar label à mensagem |
| POST | `/label/edit` | Editar label |
| POST | `/unlabel/chat` | Remover label da conversa |
| POST | `/unlabel/message` | Remover label da mensagem |
| POST | `/newsletter/create` | Criar newsletter (canal) |
| POST | `/newsletter/info` | Informações do canal |
| POST | `/newsletter/invite` | Convidar para canal |
| GET | `/newsletter/list` | Listar canais |
| POST | `/newsletter/messages` | Mensagens do canal |
| POST | `/newsletter/subscribe` | Subscrever canal |
| POST | `/community/create` | Criar comunidade |
| POST | `/community/participant/add` | Adicionar participante |
| POST | `/community/participant/remove` | Remover participante |
| POST | `/webhooks` | Criar webhook |
| GET | `/webhooks` | Listar webhooks |
| DELETE | `/webhooks/:wid` | Remover webhook |

---

## Variáveis de ambiente

| Variável | Padrão | Descrição |
|---|---|---|
| `PORT` | `8080` | Porta HTTP |
| `SERVER_HOST` | `0.0.0.0` | Host de bind |
| `API_KEY` | _(vazio)_ | Token de admin; se vazio, auth desativada |
| `LOG_LEVEL` | `info` | Nível de log (debug, info, warn, error) |
| `ENVIRONMENT` | `development` | Nome do ambiente |
| `DATABASE_URL` | `postgres://wzap:wzap123@localhost:5435/wzap?sslmode=disable` | DSN PostgreSQL |
| `NATS_URL` | `nats://localhost:4222` | Endereço NATS |
| `MINIO_ENDPOINT` | `localhost:9010` | Endereço MinIO |
| `MINIO_ACCESS_KEY` | `admin` | Usuário MinIO |
| `MINIO_SECRET_KEY` | `admin123` | Senha MinIO |
| `MINIO_BUCKET` | `wzap-media` | Bucket de mídia |
| `MINIO_USE_SSL` | `false` | TLS para MinIO |
| `WA_LOG_LEVEL` | `INFO` | Nível de log do whatsmeow |
| `GLOBAL_WEBHOOK_URL` | _(vazio)_ | Webhook global para todas as sessões |

---

## Schema do banco de dados

### `wzSessions`
| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | VARCHAR(100) PK | UUID da sessão |
| `name` | VARCHAR(100) UNIQUE | Identificador amigável (`[a-zA-Z0-9_-]+`) |
| `apiKey` | VARCHAR(255) UNIQUE | Token de autenticação da sessão |
| `jid` | VARCHAR(255) | JID WhatsApp (preenchido após pairing) |
| `qrCode` | TEXT | QR code atual (string bruta do protocolo) |
| `connected` | INTEGER | 0 = desconectado, 1 = conectado |
| `status` | VARCHAR(50) | `disconnected`, `connecting`, `connected` |
| `proxy` | JSONB | Configuração de proxy |
| `settings` | JSONB | Configurações da sessão (autoread, rejectCall, etc.) |
| `createdAt` | TIMESTAMPTZ | |
| `updatedAt` | TIMESTAMPTZ | Atualizado automaticamente via trigger |

### `wzWebhooks`
| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | VARCHAR(100) PK | UUID do webhook |
| `sessionId` | VARCHAR(100) FK | Sessão dona do webhook |
| `url` | VARCHAR(2048) | Endpoint de destino |
| `secret` | VARCHAR(255) | Segredo para HMAC-SHA256 |
| `events` | JSONB | Lista de `EventType` subscritos |
| `enabled` | BOOLEAN | Ativa/desativa entrega |
| `natsEnabled` | BOOLEAN | Usa fila NATS (com retry) em vez de HTTP direto |
| `createdAt` | TIMESTAMPTZ | |
| `updatedAt` | TIMESTAMPTZ | |

---

## Diagrama de fluxo simplificado

```
Cliente HTTP
    │
    ▼
[Fiber App]
    │
[Auth Middleware] ─── ApiKey: <token>
    │                   ├── admin token  → role=admin
    │                   └── session key → role=session
    │
[RequiredSession] ─── resolve :sessionId (nome ou UUID)
    │
[Handler] ─────────── parse body, validar input
    │
[Service] ─────────── lógica de negócio
    │
    ├── [wa.Manager] ─── GetClient(sessionID)
    │       │                 └── whatsmeow.Client
    │       │                         └── WhatsApp Protocol (TCP)
    │       │
    │       └── eventos recebidos
    │               │
    │               ├── NATS wzap.events.<sessionId>
    │               │
    │               └── webhook.Dispatcher
    │                       │
    │                       ├── HTTP POST direto
    │                       └── NATS wzap.webhook.deliver.<id>
    │                               └── consumer → HTTP POST com retry
    │
    └── [repo] ──────── PostgreSQL (sessões, webhooks)
```
