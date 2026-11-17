# Hash extra options into legacy service names

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14149](https://github.com/traefik/traefik/pull/14149) |
| Tipo | Correção |
| Área | provider/kubernetes/crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | WlnJBrn09 |
| Aberto em | 2026-10-27 (última atualização: 2026-10-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +209/-43 em 6 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/crd |
| Status | pendente |

## O que faz

Nomes legados de Service no provider CRD passam a incluir um hash das opções de cada referência (sticky, healthCheck, serversTransport). Referências ao mesmo Service com opções diferentes deixam de se sobrescrever. Sem opções extras, o nome antigo é mantido. safeNaming não muda.

## Riscos e situação no upstream

Mudança de comportamento visível (dashboard, API, métricas, access logs) e para referências cross-provider; há nota de migração. Status: needs-review. Pode quebrar quem depende dos nomes atuais.

## Principais arquivos

- `pkg/provider/kubernetes/crd/kubernetes_http.go`
- `pkg/provider/kubernetes/crd/kubernetes_test.go`
- `pkg/provider/kubernetes/crd/kubernetes_http_test.go`
- `docs/content/migrate/v3.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14149/head:upstream-pr-14149`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
