## 1. Dependência e configuração

- [x] 1.1 Atualizar `go.mau.fi/whatsmeow` para `v0.0.0-20260722203353-e9a033b24933` e sincronizar dependências com `go mod tidy`
- [x] 1.2 Adicionar `WAVersion` ao `config.Config`, carregado de `WA_VERSION` com padrão vazio
- [x] 1.3 Adaptar `Manager.DownloadMediaByPath` à nova assinatura de `DownloadMediaWithPath`, preservando o contrato interno existente

## 2. Resolução e aplicação da versão

- [x] 2.1 Implementar parser estrito de versões numéricas com suporte a `-alpha` e `-beta`
- [x] 2.2 Implementar resolução por `WA_VERSION`, `web.whatsapp.com` e fallback WPPConnect com timeouts e limite de resposta
- [x] 2.3 Aplicar a versão em `NewManager` antes do sqlstore, mantendo a versão embutida em falhas automáticas e impedindo downgrade automático
- [x] 2.4 Adicionar logs estruturados com `component=wa`, versão e fonte selecionadas

## 3. Infraestrutura e testes

- [x] 3.1 Expor e documentar `WA_VERSION` em `.env.example` e nos Composes dev/prod
- [x] 3.2 Criar testes externos de configuração para o carregamento de `WA_VERSION`
- [x] 3.3 Criar testes externos do resolver para override, sufixos, entrada inválida, fonte primária, fallback, falha total e downgrade
- [x] 3.4 Validar os Composes com `WA_VERSION` vazio e fixado

## 4. Verificação final

- [x] 4.1 Executar `gofmt`, `go mod tidy`, testes direcionados com race e build dos pacotes afetados
- [x] 4.2 Executar `go test -v -race ./...`, `go vet ./...`, `golangci-lint run ./...` e `make build`
- [x] 4.3 Validar o change com `openspec validate auto-update-whatsapp-version` e confirmar todos os artefatos completos
