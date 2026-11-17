# Scope the deny-unknown-host router to TLSRoute entry points

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14183](https://github.com/traefik/traefik/pull/14183) |
| Tipo | Correção |
| Área | provider/kubernetes/gateway |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | lazerg |
| Aberto em | 2026-10-29 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +298/-110 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/gatewayapi |
| Status | importado com ajustes |

## O que faz

O router deny-unknown-host de TLSRoute não tinha entry points, então se aplicava a todos os entry points padrão. Ele passa a usar só os entry points de listeners TLS que receberam TLSRoute. Evita fechar conexões TLS sem SNI em entry points que o Gateway não configurou.

## Riscos e situação no upstream

Muda entry points de um router gerado pelo provider Gateway API. Status: needs-review. Sem docs atualizadas.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/tlsroute.go`
- `pkg/provider/kubernetes/gateway/kubernetes_test.go`
- `pkg/provider/kubernetes/gateway/fixtures/tlsroute/with_multiple_listeners_different_ports.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14183/head:upstream-pr-14183`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Bug real: o router `deny-unknown-host`, criado quando há TLSRoute, não tinha entry points e por isso valia para todos os entry points padrão. Em entry points onde o Gateway nem tem listener TLS (ex.: `web`, um TCP qualquer, ou um entry point usado por outro provider), conexões TLS sem SNI ou com host desconhecido e ALPN fora de h2/http1.1 eram fechadas. Na prática, o provider Gateway API interferia em tráfego que não configurou.
- O PR coletava `listener.EPName`; usar os entry points do próprio router gerado é equivalente hoje e acompanha qualquer mudança futura em como o router define os seus.
- Testes do PR em conflito com o fork (nomes de routers e campos diferentes).

Ajustes:
- Entry points coletados de `router.EntryPoints` de cada router TLSRoute efetivamente mesclado.
- 30 expectativas de `deny-unknown-host` atualizadas com os entry points TLS de cada caso (nos casos mistos, o entry point `tcp` do TCPRoute sai, o que é justamente a correção); fixture nova do PR mantida e coberta por `TestLoadTLSRoutes_denyUnknownHostEntryPoints` (dois listeners TLS em entry points diferentes, com `websecure` fora).
- `integration/testdata/rawdata-gateway.json` aplicado como no PR (não validado: testes de integração não rodados).

Testes: `go test ./pkg/provider/kubernetes/gateway/` ok; `golangci-lint run ./pkg/provider/kubernetes/gateway/...` 0 issues.
