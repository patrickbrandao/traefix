# Add safeNaming option to the Kubernetes Ingress provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14157](https://github.com/traefik/traefik/pull/14157) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | amazon7737 |
| Aberto em | 2026-10-28 (última atualização: 2026-10-28) |
| Draft | não |
| Branch-alvo | `v2.11` |
| Tamanho | +293/-7 em 11 arquivos |
| Labels | status/2-needs-review, area/provider/k8s/ingress, kind/bug/fix, size/L |
| Status | pendente |

## O que faz

Adiciona a opção safeNaming ao provider Kubernetes Ingress, como já existe no CRD (PR 13689). É opt-in, então os nomes atuais de routers e services ficam iguais por padrão. Resolve colisão de nomes entre Ingresses diferentes, que hoje um sobrescreve o outro. A chave de deduplicação de certificados TLS passa a incluir o namespace.

## Riscos e situação no upstream

O PR tem base v2.11, não master. Para o fork v3.8 é preciso portar: no código local pkg/provider/kubernetes/ingress não há safeNaming hoje. Status: needs-review. Afeta nomes de routers/services e TLS.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/fixtures/Ingress-with-colliding-names-across-namespaces_ingress.yml`
- `docs/content/providers/kubernetes-ingress.md`
- `docs/content/reference/static-configuration/cli-ref.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14157/head:upstream-pr-14157`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
