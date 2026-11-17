# Fix IngressClass selection in the NGINX provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13453](https://github.com/traefik/traefik/pull/13453) |
| Tipo | Correção |
| Área | provider/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | Andrei-Predoiu |
| Aberto em | 2026-07-02 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +92/-3 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/S, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Corrige a seleção de IngressClass: quando o lookup de classes está habilitado, um Ingress que referencia classe não aceita pelo provider é ignorado, mesmo com WatchIngressWithoutClass ou anotação correspondente. Quando o RBAC impede ler IngressClass, mantém a seleção por anotação e o fallback. Relacionado a #13987.

## Riscos e situação no upstream

Status 1-needs-design-review, reviews só COMMENTED. Altera a lógica de seleção, então há risco de ignorar Ingress que deveria ser processado. Escopo pequeno (2 arquivos).

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13453/head:upstream-pr-13453`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
