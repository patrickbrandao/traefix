# Fix Docker port detection on internal networks

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13404](https://github.com/traefik/traefik/pull/13404) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | amazon7737 |
| Aberto em | 2026-06-26 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +37/-3 em 4 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/S |
| Status | pendente |

## O que faz

A detecção automática de porta no provider Docker passa a cair em Config.ExposedPorts quando NetworkSettings.Ports não tem bindings, como em containers de rede interna sem publish. A precedência fica: label de porta, bindings de runtime, exposed ports. Fixes #11878.

## Riscos e situação no upstream

Escopo pequeno (4 arquivos, +37/-3). Maintainer pediu rebase para v3.6 (branch de bugfix) e marcou need-triage. Sem review ainda. Afeta a detecção de porta, então vale testar com containers reais.

## Principais arquivos

- `pkg/provider/docker/data.go`
- `pkg/provider/docker/shared.go`
- `pkg/provider/docker/builder_test.go`
- `pkg/provider/docker/shared_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13404/head:upstream-pr-13404`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
