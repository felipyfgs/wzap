## ADDED Requirements

### Requirement: Resolução automática na inicialização

O sistema SHALL resolver a versão do cliente WhatsApp uma única vez durante a inicialização, antes de criar o sqlstore, instanciar clientes `whatsmeow` ou reconectar sessões.

#### Scenario: Fonte primária disponível

- **WHEN** `WA_VERSION` estiver vazia e `web.whatsapp.com` retornar um `client_revision` válido dentro do timeout
- **THEN** o sistema aplica a versão `2.3000.<client_revision>` antes de criar qualquer cliente WhatsApp
- **AND** registra a versão e a fonte `web.whatsapp.com` com `component=wa`

#### Scenario: Fonte primária indisponível

- **WHEN** `WA_VERSION` estiver vazia e a consulta a `web.whatsapp.com` falhar
- **THEN** o sistema consulta `currentVersion` no catálogo JSON do WPPConnect
- **AND** aceita os sufixos `-alpha` e `-beta` publicados pelo catálogo

#### Scenario: Todas as fontes indisponíveis

- **WHEN** `web.whatsapp.com` e WPPConnect falharem ou retornarem versões inválidas
- **THEN** a inicialização continua usando a versão embutida no `whatsmeow`
- **AND** um warning com `component=wa` informa a falha e a versão mantida

### Requirement: Override operacional por WA_VERSION

O sistema SHALL dar precedência absoluta à variável `WA_VERSION` quando ela contiver um valor não vazio e MUST NOT consultar fontes externas nesse caso.

#### Scenario: Override numérico válido

- **WHEN** `WA_VERSION` contiver três componentes numéricos separados por ponto, como `2.3000.1044062641`
- **THEN** o sistema aplica exatamente a versão configurada, mesmo que ela seja inferior à versão embutida
- **AND** registra a fonte `WA_VERSION`

#### Scenario: Override copiado do WPPConnect

- **WHEN** `WA_VERSION` contiver uma versão válida terminada em `-alpha` ou `-beta`
- **THEN** o sistema remove o sufixo e aplica os três componentes numéricos

#### Scenario: Override inválido

- **WHEN** `WA_VERSION` não contiver exatamente três inteiros sem sinal de 32 bits ou representar `0.0.0`
- **THEN** a criação do gerenciador WhatsApp falha antes de reconectar sessões
- **AND** o erro identifica `WA_VERSION` como configuração inválida

### Requirement: Segurança da atualização automática

O sistema SHALL limitar o tempo da resolução automática e MUST NOT aplicar automaticamente uma versão inferior à versão embutida.

#### Scenario: Proteção contra downgrade

- **WHEN** uma fonte automática retornar uma versão inferior à versão embutida
- **THEN** o sistema mantém a versão embutida
- **AND** registra um warning com as versões obtida e mantida

#### Scenario: Limite de tempo

- **WHEN** uma fonte externa não responder
- **THEN** cada requisição é limitada a cinco segundos
- **AND** o fluxo completo de resolução é limitado a dez segundos

### Requirement: Configuração em Docker Compose

Os serviços `api` dos ambientes Docker Compose de desenvolvimento e produção SHALL expor `WA_VERSION`, usando valor vazio como modo automático.

#### Scenario: Modo automático no Compose

- **WHEN** `WA_VERSION` não estiver definida no ambiente ou no arquivo `.env`
- **THEN** o container recebe valor vazio e executa a resolução automática

#### Scenario: Versão fixada no Compose

- **WHEN** o operador definir `WA_VERSION=2.3000.1044062641`
- **THEN** o container repassa esse valor sem modificação para a API

### Requirement: Ausência de efeitos em runtime

O sistema MUST NOT consultar novas versões periodicamente, desconectar clientes ou reiniciar sessões ativas para atualizar a versão.

#### Scenario: API permanece em execução

- **WHEN** a versão publicada pelo WhatsApp mudar após a inicialização
- **THEN** os clientes atuais permanecem inalterados até a próxima inicialização da API
