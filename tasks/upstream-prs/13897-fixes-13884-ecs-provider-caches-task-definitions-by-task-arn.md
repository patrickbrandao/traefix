# Fixes #13884 ECS provider caches task definitions by task ARN

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13897](https://github.com/traefik/traefik/pull/13897) |
| Tipo | Correção |
| Área | provider/ecs |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | herefindalex |
| Aberto em | 2026-09-13 (última atualização: 2026-09-30) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +113/-11 em 2 arquivos |
| Labels | status/2-needs-review, area/provider/ecs, kind/bug/fix, size/M |
| Status | pendente |

## O que faz

Passa a cachear task definitions do ECS pelo ARN da definição, e não pelo ARN da task. Tasks que compartilham a mesma revisão, incluindo as substitutas de deploy, reutilizam o resultado e evitam chamadas DescribeTaskDefinition redundantes que estouram rate limit da AWS.

## Riscos e situação no upstream

Mudança contida ao cache do provider, com testes sem credenciais AWS. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/provider/ecs/ecs.go`
- `pkg/provider/ecs/ecs_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13897/head:upstream-pr-13897`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
