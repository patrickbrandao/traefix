# Fix handle hostless ingress-nginx default backend fallback

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13445](https://github.com/traefik/traefik/pull/13445) |
| Tipo | Correção |
| Área | provider/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | amazon7737 |
| Aberto em | 2026-07-01 (última atualização: 2026-11-03) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +130/-0 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/L, area/middleware, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

Corrige o ingress-nginx quando um Ingress combina spec.defaultBackend com regra sem host. Locations de default backend sem host e sem path passam a usar PathPrefix('/') em vez de regra vazia, que quebrava o parse dos routers de fallback na v3.7.0. Esses routers recebem prioridade baixa, então rotas comuns têm precedência. Fixes #13151.

## Riscos e situação no upstream

PR base v3.7 (portar para master). Discussão ativa com maintainers sobre a abordagem de pular regras sem host no loop de default backend. Sem review aprovado. Mexe na prioridade de routers, então exige cuidado com a ordem de regras.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `pkg/provider/kubernetes/ingress-nginx/fixtures/ingresses/ingress-with-hostless-rule-and-default-backend.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13445/head:upstream-pr-13445`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- O bug foi confirmado no código atual. Com uma regra sem host e `spec.defaultBackend`, o provider ingress-nginx gerava os routers `...-rule-N-default-backend` e `-tls` com **regra vazia** (`Rule: ""`), além de um catch-all por Ingress concorrendo com o default backend global.
- A versão final do PR (pular regras sem host no loop do default backend) segue a análise do maintainer e o comportamento do ingress-nginx, em que só um Ingress sem regras substitui o backend do servidor `_`.
- O teste do PR esperava o formato antigo de chaves e metadata e não compilaria no código atual.

**Ajustes**

- Mudança aplicada manualmente em `build.go`.
- No lugar do caso gigante em `TestLoadIngresses`, criei um teste focado (`TestLoadIngresses_hostlessRuleKeepsGlobalDefaultBackend`) com uma fixture que mistura uma regra sem host e outra com host. O teste verifica que nenhum router tem regra vazia e que o fallback do default backend continua existindo para o host. Confirmei que ele falha sem a correção.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress-nginx/` ok.
- `golangci-lint run ./pkg/provider/kubernetes/ingress-nginx/...`: 0 issues.
