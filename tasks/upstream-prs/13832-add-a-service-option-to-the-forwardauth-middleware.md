# Add a service option to the ForwardAuth middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13832](https://github.com/traefik/traefik/pull/13832) |
| Tipo | Recurso |
| Área | middleware/forwardauth |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | wjabbour |
| Aberto em | 2026-09-03 (última atualização: 2026-09-27) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +2024/-47 em 24 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/docker, size/L, area/middleware, area/authentication |
| Status | pendente |

## O que faz

ForwardAuth passa a aceitar service, um service Traefik, em vez de address fixo. A requisição de auth ganha load balancing, health checks, sticky e ServersTransport (mTLS, circuit breaker), e a opção path define o caminho. Fecha #8051.

## Riscos e situação no upstream

2024 linhas em 24 arquivos, mexe em middleware, CRD e docs. address, service e tls passam a ser mutuamente exclusivos. Base master, status/2-needs-review, review não verificado.

## Principais arquivos

- `integration/forward_auth_test.go`
- `integration/resources/compose/forward_auth.yml`
- `integration/testdata/rawdata-crd.json`
- `pkg/config/dynamic/middlewares.go`
- `pkg/middlewares/auth/forward.go`
- `pkg/middlewares/auth/forward_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13832/head:upstream-pr-13832`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
