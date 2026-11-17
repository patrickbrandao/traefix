# Warn when sticky sessions are used with nativeLB

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14011](https://github.com/traefik/traefik/pull/14011) |
| Tipo | Recurso |
| Área | provider/kubernetes-crd, provider/kubernetes-ingress |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | cestercian |
| Aberto em | 2026-10-06 (última atualização: 2026-11-10) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +181/-9 em 8 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/M, area/provider/k8s/crd |
| Status | pendente |

## O que faz

Emite um warning quando sticky sessions sao configuradas em servico com nativeLB (CRD e Ingress), pois o cookie nao consegue fixar o cliente a um pod. Inclui a nota correspondente na documentacao dos dois providers.

## Riscos e situação no upstream

Apenas log e docs. Um maintainer aprovou (APPROVED) e o PR aguarda um segundo maintainer. Possui algumas mudancas em um helper compartilhado do provider CRD.

## Principais arquivos

- `pkg/provider/kubernetes/crd/kubernetes_http.go`
- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14011/head:upstream-pr-14011`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
