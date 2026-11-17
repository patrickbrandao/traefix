# Support HostRegexp as basis for TLSOptions

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11582](https://github.com/traefik/traefik/pull/11582) |
| Tipo | Recurso |
| Área | tls / snicheck / muxer |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | smerschjohann |
| Aberto em | 2025-03-04 (última atualização: 2026-04-27) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +270/-27 em 7 arquivos |
| Labels | status/1-needs-design-review, contributor/waiting-for-corrections, size/L, area/middleware, area/tls |
| Status | pendente |

## O que faz

Permite que TLSOptions sejam selecionadas por regra HostRegexp, além de Host, possibilitando opções diferentes (por exemplo sem mTLS) para domínios curinga. Resolve o caso de quem precisa de TLSOptions distintas por wildcard, algo que não era possível antes.

## Riscos e situação no upstream

Mexe no matching de TLSOptions (7 arquivos, +270/-27). Status needs-design-review e contributor/waiting-for-corrections; docs ainda não atualizadas. Sem reviews de maintainers.

## Principais arquivos

- `pkg/middlewares/snicheck/snicheck.go`
- `pkg/muxer/http/mux.go`
- `pkg/server/router/tcp/manager.go`
- `pkg/server/router/tcp/router.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11582/head:upstream-pr-11582`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
