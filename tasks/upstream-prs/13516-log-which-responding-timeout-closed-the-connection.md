# Log which responding timeout closed the connection

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13516](https://github.com/traefik/traefik/pull/13516) |
| Tipo | Recurso |
| Área | server/entrypoint (observabilidade de timeouts) |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | k-adm |
| Aberto em | 2026-07-14 (última atualização: 2026-07-29) |
| Draft | sim |
| Branch-alvo | `master` |
| Tamanho | +112/-5 em 2 arquivos |
| Labels | status/0-needs-triage, size/M |
| Status | pendente |

## O que faz

Adiciona log de debug indicando se o readTimeout ou o writeTimeout do entrypoint TCP fechou a conexão, marcado com o entrypoint. Facilita diagnosticar 502/504 que hoje não deixam claro a causa. Não muda defaults nem comportamento.

## Riscos e situação no upstream

Draft, com decisões em aberto: nível do log (o maintainer comentou que debug não ajuda e o autor concordou que não é o nível certo) e a correção sobre o idleTimeout apontada pelo maintainer. Vale esperar a versão final.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/server_entrypoint_tcp_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13516/head:upstream-pr-13516`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
