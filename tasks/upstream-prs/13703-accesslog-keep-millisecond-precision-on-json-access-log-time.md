# accesslog: keep millisecond precision on JSON access log timestamps

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13703](https://github.com/traefik/traefik/pull/13703) |
| Tipo | Recurso |
| Área | accesslogs (formato JSON) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | euginfrancis |
| Aberto em | 2026-08-17 (última atualização: 2026-08-20) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +30/-14 em 3 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/logs, size/S, area/middleware |
| Status | pendente |

## O que faz

O campo time do access log JSON tinha precisão de segundo (padrão do logrus), então requisições no mesmo segundo ficavam com timestamps idênticos, enquanto StartUTC tinha precisão maior. Agora usa formato fixo com milissegundos, só no formato JSON. Resolve #9317.

## Riscos e situação no upstream

Muda o formato do campo time, o que pode quebrar parsers que esperam RFC3339 sem fração. Status needs-review, sem review ainda.

## Principais arquivos

- `pkg/middlewares/accesslog/logger.go`
- `pkg/middlewares/accesslog/logger_formatters.go`
- `pkg/middlewares/accesslog/logger_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13703/head:upstream-pr-13703`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
