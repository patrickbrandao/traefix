# kubernetes: deduplicate backend service loads per rebuild

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13268](https://github.com/traefik/traefik/pull/13268) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress + ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | ishioni |
| Aberto em | 2026-06-02 (última atualização: 2026-06-08) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +194/-10 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/M, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Deduplica a resolução de backend (Service, EndpointSlice e anotações) dentro de cada rebuild, com chave (namespace, service, porta, e service-upstream/default-backend no ingress-nginx), em vez de executar uma vez por rota. Mesmo alvo de performance do #13011 e complementar ao #13255.

## Riscos e situação no upstream

Sobrepõe em parte o #13255 (não analisado aqui). Toca dois providers. Autor sugere dividir o PR se necessário. Status 2-needs-review, sem review; rebaseado para master.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/kubernetes_test.go`
- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/build_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13268/head:upstream-pr-13268`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
