# Bind the HTTP/3 UDP listener to udp4 only for a literal IPv4 entry point address

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13815](https://github.com/traefik/traefik/pull/13815) |
| Tipo | Correção |
| Área | http3 |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | mmskazak |
| Aberto em | 2026-08-31 (última atualização: 2026-09-03) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +72/-1 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/http3 |
| Status | pendente |

## O que faz

Em clusters IPv4 com Cilium eBPF, o HTTP/3 com endereço IPv4 literal (ex.: 0.0.0.0:8443) cria um socket só IPv6 e o tráfego QUIC IPv4 não chega. O patch usa udp4 só para IPv4 literal; os outros casos continuam com udp, que dá dual-stack. Corrige #13633.

## Riscos e situação no upstream

Pequeno (72 linhas). Base master, status/2-needs-review. Uma tentativa anterior (#13652) forçava udp6 e foi rejeitada, este PR evita esse caso.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp_http3.go`
- `pkg/server/server_entrypoint_tcp_http3_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13815/head:upstream-pr-13815`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
