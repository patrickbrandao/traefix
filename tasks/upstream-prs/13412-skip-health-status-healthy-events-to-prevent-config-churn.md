# skip health_status: healthy events to prevent config churn

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13412](https://github.com/traefik/traefik/pull/13412) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | KirylJazzSax |
| Aberto em | 2026-06-28 (última atualização: 2026-10-31) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +60/-3 em 2 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/S, area/healthcheck |
| Status | pendente |

## O que faz

Ignora eventos Docker health_status: healthy para evitar que cada um dispare rebuild da configuração. A intenção é reduzir churn em containers que já estão healthy e têm health check frequente. Relacionado ao #13321, que descreve o churn.

## Riscos e situação no upstream

Risco de regressão a verificar antes de portar. Em pkg/provider/docker/config.go:229 os containers unhealthy são filtrados, então o evento healthy é o único gatilho para reincluir um container que saiu de unhealthy. A descrição não deixa claro se o PR compara o estado anterior. Maintainer pediu rebase para v3.6.

## Principais arquivos

- `pkg/provider/docker/pdocker.go`
- `pkg/provider/docker/pdocker_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13412/head:upstream-pr-13412`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
