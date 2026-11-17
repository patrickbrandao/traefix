# Adds DTLS termination support for UDP routers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13844](https://github.com/traefik/traefik/pull/13844) |
| Tipo | Recurso |
| Área | udp, tls |
| Prioridade | **P4** |
| Valor | baixa |
| Esforço de port | alto |
| Autor | Nelwhix |
| Aberto em | 2026-09-06 (última atualização: 2026-09-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +388/-25 em 21 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/udp |
| Status | pendente |

## O que faz

Termina DTLS (RFC 6347) em routers UDP, com a opção tls que aceita apenas options nomeadas. Usa pion/dtls/v3, reaproveita o store de certificados do TLS e entrega o *dtls.Conn ao próximo handler. Ainda sem passthrough, domains ou certResolver.

## Riscos e situação no upstream

388 linhas, 21 arquivos, incluindo dependência nova (pion/dtls/v3) e reescrita da interface udp.Handler. Status status/1-needs-design-review, com o maintainer analisando impacto. Apenas um router UDP por entrypoint hoje.

## Principais arquivos

- `go.mod`
- `go.sum`
- `integration/udp_test.go`
- `pkg/config/dynamic/udp_config.go`
- `pkg/config/dynamic/zz_generated.deepcopy.go`
- `pkg/server/router/udp/router.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13844/head:upstream-pr-13844`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
