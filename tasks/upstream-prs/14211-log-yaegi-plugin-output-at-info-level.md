# Log Yaegi plugin output at info level

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14211](https://github.com/traefik/traefik/pull/14211) |
| Tipo | Recurso |
| Área | plugins/yaegi, logs |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | ShikhJohari |
| Aberto em | 2026-11-03 (última atualização: 2026-11-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +57/-1 em 3 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/logs, size/S, area/plugins |
| Status | pendente |

## O que faz

A saída de fmt.Print*, print e println dos plugins Yaegi passa a ser logada em INFO, em vez de DEBUG. Assim, plugins conseguem reportar o que fazem sem ligar o DEBUG. Stderr do interpretador continua em ERROR. Migration guide do v3.9 ganha uma nota.

## Riscos e situação no upstream

Muda o nível de log de plugins que imprimem muito; pode aumentar volume em INFO. Status: needs-review, base master. Escopo pequeno.

## Principais arquivos

- `pkg/plugins/middlewareyaegi.go`
- `pkg/plugins/middlewareyaegi_test.go`
- `docs/content/migrate/v3.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14211/head:upstream-pr-14211`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
