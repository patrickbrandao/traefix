# Report node addresses in Ingress status for Ingress NGINX provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13839](https://github.com/traefik/traefik/pull/13839) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | masharma2004 |
| Aberto em | 2026-09-05 (última atualização: 2026-10-26) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +365/-8 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/provider/k8s, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Sem LoadBalancer nem publish-service, o ingress-nginx publica os IPs dos nós no status do Ingress. Este PR faz o mesmo: descobre os pods do controller, resolve os nós e publica os endereços externos (ou internos com reportNodeInternalIPAddress). Útil em bare-metal com DaemonSet, onde o external-dns não tem o que publicar.

## Riscos e situação no upstream

365 linhas em 6 arquivos. Exige RBAC de pods e nodes list, testado no startup; sem a permissão, o fallback fica desligado. Base master, status/2-needs-review, review não verificado.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/client.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `docs/content/reference/dynamic-configuration/kubernetes-ingress-nginx-rbac.yml`
- `docs/content/reference/install-configuration/configuration-options.md`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13839/head:upstream-pr-13839`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
