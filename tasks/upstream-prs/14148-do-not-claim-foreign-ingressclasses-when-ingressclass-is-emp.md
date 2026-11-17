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
| Status | rejeitado |

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

## Revisão (2026-11-16)

**Achados**

- O PR muda o comportamento padrão do provider **Kubernetes Ingress** (e não do ingress-nginx): com `providers.kubernetesIngress.ingressClass` vazio, hoje são processados os Ingresses de qualquer IngressClass cujo controller é `traefik.io/ingress-controller`. Com o PR, só a IngressClass com o nome literal `traefik` seria processada.
- É uma **mudança incompatível** numa versão de correção: instalações com IngressClass de outro nome (por exemplo, `traefik-lb`, `traefik-internal`, ou o nome gerado pelo Helm chart) e `ingressClass` vazio deixariam de rotear tudo de uma vez, sem nenhum erro. O próprio PR precisa de uma seção de migração e altera fixtures e testes existentes que hoje passam.
- O problema que o PR quer resolver (duas instâncias do Traefik reivindicando as mesmas IngressClasses) já tem solução configurável: definir `ingressClass` em cada instância. As IngressClasses envolvidas pertencem ao mesmo controller por definição, então não há cruzamento de fronteira de segurança.
- No upstream só passou por revisão automática, sem decisão de maintainer.

**Decisão**

Rejeitado: quebra compatibilidade sem ganho de segurança. O problema parecido no provider ingress-nginx (um Ingress de **outro** controller sendo reivindicado) foi corrigido com #13453.

**Testes**

Nenhum (nada importado).
