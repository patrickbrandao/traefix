# Redact static provider fields

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14305](https://github.com/traefik/traefik/pull/14305) |
| Tipo | Correção |
| Área | logs/redactor (providers docker, swarm, http, knative) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jspdown |
| Aberto em | 2026-11-16 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +585/-4 em 6 arquivos |
| Labels | status/0-needs-triage, area/provider/docker, kind/bug/fix, size/M, area/provider/docker/swarm, area/provider/http, area/provider/k8s/knative |
| Status | pendente |

## O que faz

Campos sensíveis de providers (password do Docker/Swarm, token do Knative e headers do HTTP) não tinham a tag loggable:"false" e apareciam no log DEBUG da configuração estática. O PR adiciona a tag e remove export:"true" dos headers do provider HTTP.

## Riscos e situação no upstream

Base é v3.7, precisa de port. Remover export:"true" dos headers do HTTP muda o resultado de Anonymize, então a saída de config muda. Os arquivos de testdata podem conflitar com #14304. Sem review de maintainer ainda.

## Principais arquivos

- `pkg/provider/docker/shared.go`
- `pkg/provider/http/http.go`
- `pkg/provider/kubernetes/knative/kubernetes.go`
- `pkg/redactor/redactor_config_test.go`
- `pkg/redactor/testdata/anonymized-static-config.json`
- `pkg/redactor/testdata/secured-static-config.json`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14305/head:upstream-pr-14305`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
