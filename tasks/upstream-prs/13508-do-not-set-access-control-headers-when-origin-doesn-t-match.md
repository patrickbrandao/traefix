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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

**Veredito: importado com ajustes** (ajustes pequenos).

Análise:
- Fix correto e alinhado ao Fetch Standard e à recomendação CORS do W3C. `Access-Control-Allow-Credentials` e `Access-Control-Expose-Headers` só são enviados quando o `Origin` casa com a lista permitida. No preflight, um `Origin` que não casa recebe só `200` com `Content-Length: 0`, sem nenhum `Access-Control-*`.
- **Endurecimento de segurança**: antes, `Access-Control-Allow-Credentials: true` e a lista de headers expostos eram enviados para qualquer origem, inclusive em requisições sem `Origin`, o que revelava a política para terceiros. O navegador já bloqueava o acesso (falta `Allow-Origin`), então não havia bypass, só ruído e vazamento de configuração.
- O `Vary: Origin` continua sendo adicionado mesmo quando a origem não casa, o que é necessário para caches. Verifiquei a interação com o #12222 (já importado).
- Sem review de maintainer ainda. O PR é pré-requisito do #13499 (conformance Gateway API).

Ajustes:
- Não importada a mudança em `integration/gateway_api_conformance_test.go` (`TestIsolation` do conformance). Ela não tem relação com o middleware, só serve para expor o bug no upstream, e não pôde ser validada aqui (requer Docker/k3s).
- Novo caso de teste: origem que não casa, com `addVaryHeader`, ainda recebe `Vary: Origin` e nenhum `Access-Control-*`.

Testes:
- `go test ./pkg/middlewares/headers/` ok.
- `golangci-lint run ./pkg/middlewares/headers/...`: 0 issues.
- `go build ./...` ok.

## Integração (2026-11-17)

Na integração com o #14133 (g08), o modo `CORSStrict` foi combinado com esta correção (detalhes na task 14133).
