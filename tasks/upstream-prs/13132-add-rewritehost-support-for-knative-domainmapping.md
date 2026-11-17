# Add rewriteHost support for Knative DomainMapping

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13132](https://github.com/traefik/traefik/pull/13132) |
| Tipo | Recurso |
| Área | provider/kubernetes/knative |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | Abirdcfly |
| Aberto em | 2026-05-09 (última atualização: 2026-09-22) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +242/-17 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/provider/k8s, area/provider/k8s/knative |
| Status | pendente |

## O que faz

Implementa o rewriteHost que estava como TODO no provider Knative. Quando um path do Knative Ingress define rewriteHost, como no DomainMapping, cria um middleware Headers que reescreve o Host e adiciona K-Original-Host para o backend. Resolve o issue #13131.

## Riscos e situação no upstream

Escopo restrito a quem usa Knative. Label needs review sem revisão de maintainer; autor pediu ajuda para avançar. Toca o provider Knative e fixtures.

## Principais arquivos

- `pkg/provider/kubernetes/knative/kubernetes.go`
- `pkg/provider/kubernetes/knative/kubernetes_test.go`
- `pkg/provider/kubernetes/knative/fixtures/domain_mapping.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13132/head:upstream-pr-13132`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
