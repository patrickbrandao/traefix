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
| Status | importado |

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

## Revisão (2026-11-16)

**Achados no PR original**

- O bug é real e afeta isolamento entre controllers: `shouldProcess` só respeitava `spec.ingressClassName` quando a lista filtrada de IngressClasses **não estava vazia**. Se nenhuma IngressClass do cluster casasse com o provider (por exemplo, o cluster só tem a classe `traefik` do provider Kubernetes Ingress), um Ingress com `ingressClassName: traefik` caía na lógica de annotation e `watchIngressWithoutClass`. Com `watchIngressWithoutClass: true`, o provider ingress-nginx passava a publicar rotas de Ingresses que pertencem a outro controller.
- A correção usa `ignoreIngressClasses` (só fica verdadeiro quando o RBAC proíbe listar IngressClasses) em vez de "lista vazia". Com isso, o comportamento sem permissão de cluster continua como antes.
- O maintainer aprovou ("Overall, LGTM").

**Ajustes**

- Nenhum ajuste funcional. Os literais do teste foram adaptados ao lint `embedlit`.
- Relacionado: #14148 trata uma assimetria parecida no provider Kubernetes Ingress (rejeitado; veja a task dele).

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress-nginx/` ok (novo `TestProvider_shouldProcess`).
- `golangci-lint run ./pkg/provider/kubernetes/ingress-nginx/...`: 0 issues.
