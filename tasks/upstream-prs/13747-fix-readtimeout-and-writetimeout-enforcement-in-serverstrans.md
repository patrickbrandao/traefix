# Fix readTimeout and writeTimeout enforcement in ServersTransport

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13747](https://github.com/traefik/traefik/pull/13747) |
| Tipo | Correção |
| Área | server/servers-transport, provider/kubernetes/crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | mmatur |
| Aberto em | 2026-08-21 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +411/-170 em 12 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/L, area/server, area/provider/k8s/crd, area/provider/k8s |
| Status | pendente |

## O que faz

Corrige regressões do enforcement de readTimeout/writeTimeout no FastProxy (introduzido no #12716): os deadlines passam a ser armados só durante escrita de request e leitura de response, então conexões ociosas e WebSocket não são mais derrubadas e um request não pode mais pendurar. Também documenta o comportamento com HTTP/2.

## Riscos e situação no upstream

Afeta FastProxy (experimental) e o ingress-nginx com fastProxy. Base master, status/2-needs-review, review não verificado. Depende do #12716 já estar no fork. Toca 12 arquivos, incluindo CRD.

## Principais arquivos

- `pkg/proxy/fast/builder.go`
- `pkg/proxy/fast/builder_test.go`
- `pkg/proxy/fast/connpool.go`
- `pkg/proxy/fast/connpool_test.go`
- `pkg/proxy/fast/proxy.go`
- `pkg/proxy/fast/proxy_websocket_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13747/head:upstream-pr-13747`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
