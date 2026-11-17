# Make ForwardAuth middleware timeout configurable

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14024](https://github.com/traefik/traefik/pull/14024) |
| Tipo | Recurso |
| Área | middleware/forwardAuth |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | bhaskargurram-ai |
| Aberto em | 2026-10-08 (última atualização: 2026-10-09) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +192/-1 em 18 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/authentication, area/provider/k8s/crd |
| Status | pendente |

## O que faz

Adiciona a opcao timeout ao middleware ForwardAuth, que antes era fixo em 30s. Padrao 30s (sem mudanca), 0 desabilita, e em caso de estouro responde 500. Funciona em file, labels, KV e CRD.

## Riscos e situação no upstream

Logica pequena (+192/-1), mas toca varios arquivos de CRD gerado e docs. Sem reviews de maintainer; label status/2-needs-review. Resolve #9857.

## Principais arquivos

- `pkg/middlewares/auth/forward.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`
- `docs/content/reference/routing-configuration/http/middlewares/forwardauth.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14024/head:upstream-pr-14024`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
