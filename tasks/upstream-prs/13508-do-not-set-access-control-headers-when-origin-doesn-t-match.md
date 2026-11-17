# Do not set Access-Control-* headers when Origin doesn't match

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13508](https://github.com/traefik/traefik/pull/13508) |
| Tipo | Correção |
| Área | middleware/headers (CORS) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | gndz07 |
| Aberto em | 2026-07-10 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +67/-15 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

O middleware de CORS passa a enviar os headers Access-Control-* apenas quando o Origin da requisição bate com uma origem permitida, inclusive em preflight, conforme Fetch Standard e W3C. Hoje esses headers são enviados sem essa checagem. Também fecha falha de conformidade com os testes de Gateway API.

## Riscos e situação no upstream

Muda comportamento para origens não listadas (os headers deixam de ser enviados). PR base v3.7, o que exige port para a linha de manutenção. Status needs-review, sem review de maintainer ainda.

## Principais arquivos

- `pkg/middlewares/headers/header.go`
- `pkg/middlewares/headers/header_test.go`
- `integration/gateway_api_conformance_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13508/head:upstream-pr-13508`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
