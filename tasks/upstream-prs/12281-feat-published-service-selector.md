# Feat/published service selector

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12281](https://github.com/traefik/traefik/pull/12281) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | justin0u0 |
| Aberto em | 2025-11-16 (última atualização: 2026-01-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +315/-71 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/L |
| Status | pendente |

## O que faz

Adiciona a opção --publishedserviceselector para escolher e publicar IPs de vários Services, resolvendo a issue 11617.

## Riscos e situação no upstream

Nova opção de CLI e mudança no provider Ingress (6 arquivos, +315/-71). Status needs-review, sem reviews.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/client.go`
- `docs/content/reference/install-configuration/configuration-options.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12281/head:upstream-pr-12281`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
