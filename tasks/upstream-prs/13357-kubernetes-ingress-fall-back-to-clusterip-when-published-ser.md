# Kubernetes Ingress: fall back to ClusterIP when published service has no ExternalIPs

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13357](https://github.com/traefik/traefik/pull/13357) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | goingforstudying-ctrl |
| Aberto em | 2026-06-12 (última atualização: 2026-11-03) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +322/-1 em 8 arquivos |
| Labels | status/2-needs-review, area/provider/k8s/ingress, kind/bug/fix, size/M |
| Status | adiado (revertido após revisão independente) |

## O que faz

Quando publishedService aponta para um Service ClusterIP sem externalIPs, o status do Ingress fica vazio, e ferramentas como ArgoCD consideram o Ingress incompleto. O PR usa o ClusterIP como endereço do LB (exceto None), como o nginx-ingress faz. Relacionado à issue #13324.

## Riscos e situação no upstream

PR base v3.7 (portar para master). Há discussão com usuários e maintainers sobre a abordagem: o PR #12134 propõe trocar para ClusterIP, enquanto este preserva ExternalIPs e faz fallback. Precisa alinhar antes. Publicar ClusterIP pode ser enganoso se não for alcançável de fora. Sem review.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/kubernetes_test.go`
- `pkg/provider/kubernetes/ingress/fixtures/Published-Service-ClusterIP-No-ExternalIPs.yml`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-ingress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13357/head:upstream-pr-13357`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- A abordagem é melhor que a de #12134: os `externalIPs` continuam publicados quando existem, e o ClusterIP só entra como fallback. Assim o Ingress deixa de ficar sem status, que era o que travava o Argo CD em "Progressing".
- Limitação: usava só `spec.clusterIP`; em Services dual-stack a segunda família (`spec.clusterIPs[1]`) era ignorada.
- Risco a documentar (não é bug): quem usa External-DNS com um published Service ClusterIP sem `externalIPs` antes não tinha status e agora vai publicar um IP **interno** do cluster no DNS. A documentação do PR já avisa sobre isso.

**Ajustes**

- Combinado com a ideia de #12134: o fallback usa `spec.clusterIPs` (as duas famílias), cai para `spec.clusterIP` se a lista vier vazia e ignora `""` e `None` (`corev1.ClusterIPNone`).
- Adaptado à função `serviceIngressStatus` (introduzida na importação de #12281), o que também cobre o `publishedServiceSelector`.
- Fixture e caso de teste novos para dual-stack; documentação ajustada.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress/` ok (casos No-ExternalIPs, IPv6, Dual Stack, Headless, Empty).
- `golangci-lint run ./pkg/provider/kubernetes/ingress/...`: 0 issues.

## Revisão independente pós-integração (2026-11-17)

O import foi **revertido**. Antes, um `publishedService` do tipo ClusterIP sem `externalIPs` deixava o status do Ingress vazio. O fallback passa a publicar os ClusterIPs (endereços privados, como 10.x) nesse status. Com o external-dns lendo o status do Ingress, isso gera registros DNS públicos apontando para IPs internos. O efeito é vazar o endereçamento interno e mudar o comportamento de quem conta com o status vazio.

Reimportar só como opção explícita, desligada por padrão (por exemplo `ingressEndpoint.publishClusterIPs`). O suporte dual-stack (`clusterIPs`) só faz sentido junto com essa opção.
