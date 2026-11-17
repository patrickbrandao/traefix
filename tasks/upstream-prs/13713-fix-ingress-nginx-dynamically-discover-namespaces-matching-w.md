# fix(ingress-nginx): dynamically discover namespaces matching watchNamespaceSelector

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13713](https://github.com/traefik/traefik/pull/13713) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | luisdeniz |
| Aberto em | 2026-08-18 (última atualização: 2026-10-23) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +542/-1 em 3 arquivos |
| Labels | status/2-needs-review, size/L, area/provider/k8s, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Com watchNamespaceSelector, o provider ingress-nginx hoje só descobre namespaces que casam com o seletor na inicialização. Esta mudança adiciona um informer de Namespace que inicia/para informers por namespace conforme labels mudam, sem precisar reiniciar o pod.

## Riscos e situação no upstream

Base v3.7, então precisa de adaptação para master. Status status/2-needs-review, sem review de maintainer ainda. Toca o mesmo tema de seleção de namespaces do #13806. Adiciona watch cluster-scoped de Namespace (exige RBAC de namespaces).

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/client.go`
- `pkg/provider/kubernetes/ingress-nginx/client_test.go`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13713/head:upstream-pr-13713`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
