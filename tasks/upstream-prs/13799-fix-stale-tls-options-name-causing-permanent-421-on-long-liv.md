# Fix stale TLS options name causing permanent 421 on long-lived connections

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13799](https://github.com/traefik/traefik/pull/13799) |
| Tipo | Correção |
| Área | middleware/tls, provider/kubernetes |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | sk-sanjay01 |
| Aberto em | 2026-08-28 (última atualização: 2026-09-14) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +259/-0 em 5 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware, area/tls, area/provider/k8s |
| Status | pendente |

## O que faz

Conexões keep-alive resolvem o nome das TLS options uma vez, no accept. Se o router chegou depois, o request cai em 421 para sempre nessa conexão. O patch adiciona Connection: close na resposta 421 do snicheck, forçando o cliente a rehandshake e reavaliar as options.

## Riscos e situação no upstream

Base v3.7, precisa de adaptação. Trata o sintoma, não a causa (o nome das options fica fixo por conexão). Corrige #13745. Status/2-needs-review.

## Principais arquivos

- `pkg/middlewares/snicheck/snicheck.go`
- `pkg/middlewares/snicheck/snicheck_test.go`
- `pkg/server/server_entrypoint_tcp_http3.go`
- `pkg/server/server_entrypoint_tcp_test.go`
- `pkg/tcp/tls.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13799/head:upstream-pr-13799`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
