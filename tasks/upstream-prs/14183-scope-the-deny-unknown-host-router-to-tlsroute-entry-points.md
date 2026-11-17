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
| Status | pendente |

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
