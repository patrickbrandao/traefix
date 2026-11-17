# feat: write status conditions to Traefik CRDs after config load

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13285](https://github.com/traefik/traefik/pull/13285) |
| Tipo | Recurso |
| Área | provider/kubernetes/crd |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | Alex-Waring |
| Aberto em | 2026-06-03 (última atualização: 2026-09-11) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +4064/-49 em 62 arquivos |
| Labels | status/1-needs-design-review, size/L, area/provider/k8s/crd, area/provider/k8s |
| Status | pendente |

## O que faz

Após carregar a configuração, escreve condições de status em todos os recursos do provider CRD do Traefik, para que quem consulta a API do Kubernetes veja se a rota foi carregada. A escrita é não bloqueante: falha gera só warning. Inclui RBAC novo no Helm chart. Issue de origem: #10015.

## Riscos e situação no upstream

Enorme: 62 arquivos, +4064 linhas, com regeneração de CRDs e manifests. Adiciona permissão de escrita em status de todos os recursos. Status 1-needs-design-review; maintainers responderam que está no radar, sem ETA, e há cobranças sem resposta.

## Principais arquivos

- `pkg/provider/kubernetes/crd/`
- `docs/content/reference/dynamic-configuration/kubernetes-crd-rbac.yml`
- `docs/content/reference/dynamic-configuration/traefik.io_ingressroutes.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13285/head:upstream-pr-13285`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
