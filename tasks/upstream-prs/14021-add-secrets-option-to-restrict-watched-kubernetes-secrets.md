# Add secrets option to restrict watched Kubernetes Secrets

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14021](https://github.com/traefik/traefik/pull/14021) |
| Tipo | Recurso |
| Área | provider/kubernetes-ingress, provider/kubernetes-crd, provider/kubernetes-gateway |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | Tolsto |
| Aberto em | 2026-10-07 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +917/-73 em 18 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/L, area/provider/k8s/crd, area/provider/k8s, area/provider/k8s/gatewayapi |
| Status | adiado |

## O que faz

Adiciona a opcao secrets (namespace -> lista de nomes) aos providers Kubernetes, com um informer por Secret, para o Traefik nao precisar de acesso a todos os Secrets do namespace. Namespace com * observa todos os Secrets daquele namespace.

## Riscos e situação no upstream

Grande (+917/-73, 18 arquivos) e mexe nos tres providers. Um maintainer comentou que list/watch no RBAC padrao nao e restringido por resourceNames, premissa da descricao; o ganho de least privilege precisa ser revalidado. Outro maintainer pediu suporte ao kubernetesIngressNGINX. Label status/2-needs-review; so um review COMMENTED.

## Principais arquivos

- `pkg/provider/kubernetes/k8s/secrets.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`
- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `pkg/provider/kubernetes/ingress/client.go`
- `pkg/provider/kubernetes/crd/client.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14021/head:upstream-pr-14021`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados**

- A ideia vale do ponto de vista de segurança (menor privilégio): a opção `secrets` limita quais Secrets o Traefik observa. O motivo principal do PR, porém, não se sustenta como foi documentado:
  - O PR diz que "o RBAC pode ser restrito com `resourceNames`". Informers só fazem `list`/`watch`, e o RBAC padrão do Kubernetes **não** aplica `resourceNames` a `list`/`watch` (só a `get`), a menos que o cluster tenha o *authorize with selectors* (KEP-4601) ativo. Sem isso, um Role restrito por `resourceNames` recebe `Forbidden` no `list`/`watch` com field selector, e o provider **não sobe**. A documentação adicionada não menciona essa exigência. O revisor automático apontou o mesmo problema e não houve resposta.
  - Sem RBAC restrito, o ganho fica só na memória/cache local: o Traefik continua com permissão de ler todas as Secrets.
- O escopo é amplo: muda o cliente de três providers (Kubernetes Ingress, CRD e Gateway), substitui os mapas `factoriesSecret` por um helper novo em `pkg/provider/kubernetes/k8s` e adiciona fixtures de integração. CRD e Gateway ficam fora do foco deste grupo e são mexidos por outras importações, então o risco de conflito é alto.
- Mudanças periféricas sem relação com o PR: newline final em `pkg/plugins/fixtures/src/*/go.mod` (inofensivo).
- Pedido aberto no PR para suportar também o ingress-nginx, que não está incluído.
- Falta teste para o cenário principal (vários namespaces observados, cada um com sua lista), como o revisor apontou.
- Nenhum código suspeito ou dependência nova.

**Motivo do adiamento**

Recurso grande, sem revisão de maintainer, cobrindo três providers e com documentação de RBAC incorreta. Pode quebrar a inicialização de quem seguir a doc. Reavaliar quando o upstream fechar o design (incluindo o ingress-nginx e a doc de RBAC, que exige `AuthorizeWithSelectors`).

**Testes**

Nenhum (nada importado).
