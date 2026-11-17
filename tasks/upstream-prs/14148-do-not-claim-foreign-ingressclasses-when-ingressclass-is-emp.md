# Do not claim foreign IngressClasses when ingressClass is empty

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14148](https://github.com/traefik/traefik/pull/14148) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | WlnJBrn09 |
| Aberto em | 2026-10-27 (última atualização: 2026-10-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +36/-18 em 5 arquivos |
| Labels | status/2-needs-review, area/provider/k8s/ingress, kind/bug/fix, size/S |
| Status | pendente |

## O que faz

Com providers.kubernetesIngress.ingressClass vazio, o provider Ingress passa a casar só com a IngressClass chamada traefik, como já faz para a anotação. Assim, uma segunda instância que observa outra IngressClass não duplica rotas nem brigar pelo status. Alinha com o provider CRD.

## Riscos e situação no upstream

Muda comportamento para quem usa ingressClass vazio com spec.ingressClassName diferente de traefik. Requer migração (já documentada em migrate/v3.md). Status: needs-review. Escopo pequeno.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/kubernetes_test.go`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-ingress.md`
- `docs/content/migrate/v3.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14148/head:upstream-pr-14148`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
