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
| Status | importado |

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

## Revisão (2026-11-16)

### Achados

- Sem review de maintainer; correção pequena (fixes #9317): o campo `time` do access log JSON usava o default do logrus (RFC3339, segundos) e requisições no mesmo segundo ficavam com timestamps iguais.
- O formato novo (`2006-01-02T15:04:05.000Z07:00`) continua RFC3339 válido, com largura fixa; parsers que esperam RFC3339 seguem funcionando. Só o formato `json` muda.
- Nada de segurança envolvido.

### Ajustes

Nenhum: aplicado como está (o PR já reforça os testes existentes do formatter JSON).

### Testes

- `go test ./pkg/middlewares/accesslog/`: ok.
- `golangci-lint run ./pkg/middlewares/accesslog/...`: 0 issues.
