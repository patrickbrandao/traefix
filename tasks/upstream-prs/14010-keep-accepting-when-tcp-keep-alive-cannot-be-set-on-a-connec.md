# Keep accepting when TCP keep-alive cannot be set on a connection

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14010](https://github.com/traefik/traefik/pull/14010) |
| Tipo | Correção |
| Área | server/entrypoint-tcp |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | headw01 |
| Aberto em | 2026-10-05 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +108/-8 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/server |
| Status | pendente |

## O que faz

Quando o peer reseta a conexao entre accept() e a configuracao de keep-alive, o erro saia como erro do listener e o entrypoint TCP parava de aceitar conexoes de vez (macOS, illumos, FreeBSD). Agora a conexao e fechada e o listener segue aceitando.

## Riscos e situação no upstream

Pequeno (+108/-8). Duplicado funcional do #14094 (mesmo problema, mesmas issues #8841/#8071). Label status/design-review; dois reviews COMMENTED; autor cobrou andamento.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/server_entrypoint_tcp_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14010/head:upstream-pr-14010`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
