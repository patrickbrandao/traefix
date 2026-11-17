# Improve middleware missing errors for double-dash Kubernetes names

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13877](https://github.com/traefik/traefik/pull/13877) |
| Tipo | Recurso |
| Área | provider/kubernetes-ingress, middleware-naming |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | harriteja |
| Aberto em | 2026-09-09 (última atualização: 2026-09-14) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +142/-2 em 6 arquivos |
| Labels | status/1-needs-design-review, area/provider/k8s/ingress, size/M, area/provider/k8s |
| Status | pendente |

## O que faz

Adiciona aviso quando uma referência de middleware em annotation de Ingress/Service tem caracteres não alfanuméricos consecutivos que provider.Normalize colapsa, e enriquece o erro middleware does not exist com a forma normalizada. A normalização em si não muda, só o diagnóstico e a documentação.

## Riscos e situação no upstream

Não altera normalização (seria breaking), então é não-disruptivo. Status design-review, sem review de maintainer ainda. Toca 6 arquivos, incluindo pkg/provider/configuration.go e os middlewares HTTP e TCP.

## Principais arquivos

- `pkg/provider/configuration.go`
- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/server/middleware/middlewares.go`
- `pkg/server/middleware/tcp/middlewares.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13877/head:upstream-pr-13877`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
