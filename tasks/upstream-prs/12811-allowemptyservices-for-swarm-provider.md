# AllowEmptyServices for Swarm provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12811](https://github.com/traefik/traefik/pull/12811) |
| Tipo | Correção |
| Área | provider/docker/swarm |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | clifmo |
| Aberto em | 2026-03-14 (última atualização: 2026-09-21) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +239/-10 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/provider/docker/swarm |
| Status | pendente |

## O que faz

A opção AllowEmptyServices existia na configuração do provider Swarm mas não funcionava. Após a correção, services escalados para zero são mantidos e não respondem 404, o que permite que o Sablier acorde o service sob demanda. Segue o padrão do provider Docker e o padrão é false.

## Riscos e situação no upstream

Pequeno (3 arquivos, com testes para true e false). Aprovado por maintainer. O PR é contra v3.7 (maintenance), então precisa de backport ou adaptação. Docs ainda não atualizadas.

## Principais arquivos

- `pkg/provider/docker/config.go`
- `pkg/provider/docker/pswarm.go`
- `pkg/provider/docker/pswarm_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12811/head:upstream-pr-12811`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
