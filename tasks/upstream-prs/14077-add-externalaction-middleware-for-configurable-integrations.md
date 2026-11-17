# Add ExternalAction middleware for configurable integrations

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14077](https://github.com/traefik/traefik/pull/14077) |
| Tipo | Recurso |
| Área | middleware/externalAction, provider/kubernetes-crd |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | bjzhangshuai |
| Aberto em | 2026-10-15 (última atualização: 2026-10-23) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +17025/-27 em 79 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware, area/provider/k8s/crd |
| Status | pendente |

## O que faz

Middleware HTTP configuravel que observa requisicao/resposta, mapeia campos (JSON, form, headers, query) e chama um servico externo via HTTP ou executavel, de forma sincrona ou assincrona, com dedup opcional em memoria ou MySQL e validacao de resposta.

## Riscos e situação no upstream

Enorme (+17025/-27, 79 arquivos, go.mod novo). Label status/1-needs-design-review; mais de 30 reviews COMMENTED. Executa binarios externos e faz chamadas HTTP saindo do proxy, o que exige analise de seguranca profunda. Escopo amplo demais para port direto.

## Principais arquivos

- `pkg/config/dynamic/external_action.go`
- `pkg/config/dynamic/middlewares.go`
- `go.mod`
- `docs/content/reference/routing-configuration/http/middlewares/externalaction.md`
- `integration/externalaction_security_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14077/head:upstream-pr-14077`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
