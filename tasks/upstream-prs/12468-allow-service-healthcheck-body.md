# Allow service healthcheck body

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12468](https://github.com/traefik/traefik/pull/12468) |
| Tipo | Recurso |
| Área | healthcheck |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | aek |
| Aberto em | 2025-12-24 (última atualização: 2026-03-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +44/-1 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, contributor/waiting-for-corrections, size/S, area/healthcheck |
| Status | importado com ajustes |

## O que faz

Permite definir um body na healthcheck de services HTTP, para usar requisições POST, por exemplo JSON-RPC (caso Odoo). Inclui exemplos na documentação.

## Riscos e situação no upstream

Pequeno (+44/-1). Um reviewer pediu mudanças (CHANGES_REQUESTED) e o autor ainda não fez correções (contributor/waiting-for-corrections).

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/healthcheck/healthcheck.go`
- `docs/content/routing/services/index.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12468/head:upstream-pr-12468`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:

- A mudança é pequena e correta. `strings.NewReader` define `GetBody` e `ContentLength`, então o corpo é reenviado em redirects 307/308 e um novo reader é criado a cada checagem (o corpo não se "esgota").
- Um maintainer pediu testes unitários (`CHANGES_REQUESTED`), que nunca chegaram.
- A documentação foi aplicada em caminhos antigos que não existem neste código.
- O campo `Body` não tem `export:"true"`. Mantido assim de propósito: o corpo pode conter credenciais (ex.: JSON-RPC) e não deve sair em dumps anonimizados.
- O campo não existe no CRD Kubernetes (`traefikio/v1alpha1.ServerHealthCheck`), igual ao PR original. Ele só vale para file, KV e labels.

Ajustes:

- Testes novos. Um caso `custom body` em `TestServiceHealthChecker_newRequest` confere corpo e `ContentLength`. E um `TestServiceHealthChecker_checkHealthHTTP_Body` ponta a ponta confere método, `Content-Type` e corpo recebidos pelo servidor em duas checagens seguidas.
- Docs: `body` incluído em `reference/dynamic-configuration/file.{toml,yaml}` e na tabela de opções de `reference/routing-configuration/http/load-balancing/service.md` (com a nota de que é ignorado em `mode: grpc`).

Testes:

- `go build ./...`: ok.
- `go test ./pkg/healthcheck/ ./pkg/config/...`: ok. Os testes novos passaram com `-race`.
- `golangci-lint run ./pkg/healthcheck/... ./pkg/config/dynamic/...`: 0 issues.
- Observação: `go test -race ./pkg/healthcheck/` já falhava antes deste PR em `TestDifferentIntervals`. A causa é uma race no mock `testhelpers.CollectingGauge`, que não é seguro entre goroutines e recebe escritas das goroutines healthy e unhealthy. Isso foi tratado na importação do #12485.
