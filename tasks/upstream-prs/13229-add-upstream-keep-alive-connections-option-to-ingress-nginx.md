# Add upstream keep-alive connections option to ingress-nginx provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13229](https://github.com/traefik/traefik/pull/13229) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | amazon7737 |
| Aberto em | 2026-05-22 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +108/-14 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Adiciona a opção upstreamKeepaliveConnections ao provider Kubernetes Ingress NGINX, mapeada para ServersTransport.MaxIdleConnsPerHost (padrão 320, igual ao ingress-nginx). Sem isso, os transports gerados caem no padrão do Go (2 conexões ociosas por host). Fecha a lacuna do par com upstreamKeepaliveTimeout. Fixes #13123.

## Riscos e situação no upstream

Sem review dos maintainers (só comentário 'Any update?'); status 2-needs-review. Mudança pequena e isolada, inclui testes e docs. Sem conflito conhecido.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13229/head:upstream-pr-13229`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
