# fix(middleware/headers): preserve existing Vary headers when adding Origin

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12222](https://github.com/traefik/traefik/pull/12222) |
| Tipo | Correção |
| Área | middleware/headers |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | ris-tlp |
| Aberto em | 2025-10-31 (última atualização: 2026-09-04) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +3/-2 em 1 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/S, area/middleware |
| Status | importado com ajustes |

## O que faz

Ao adicionar Vary: Origin, o middleware headers lia só o primeiro valor de Vary com Header.Get e perdia os demais. A correção lê o slice completo de res.Header["Vary"]. Bug confirmado localmente em pkg/middlewares/headers/header.go linha 103.

## Riscos e situação no upstream

Mudança de três linhas, bem localizada. Status design-review, sem aprovação. Autor cobra review há meses.

## Principais arquivos

- `pkg/middlewares/headers/header.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12222/head:upstream-pr-12222`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:
- O bug procede. `res.Header.Get("Vary")` lê só a primeira linha de `Vary`, e o `Set` seguinte apagava as outras. Com backend que manda `Vary` em várias linhas (ou com compress na frente), valores como `Cookie` e `Authorization` sumiam, e um cache podia servir a variante errada.
- O fix proposto usa `strings.Contains(varyHeader, "Origin")`, que é frouxo: um header chamado `X-Origin-Id` era tomado por `Origin`, e o `Vary: Origin` deixava de ser adicionado. A comparação também diferencia maiúsculas, e nomes de header não diferenciam.
- O PR não traz testes. Continua em `status/1-needs-design-review`, sem objeção técnica de maintainer.
- Nenhum problema de segurança além do envenenamento de cache já descrito, que é o próprio bug.

Ajustes:
- Os valores de `Vary` são lidos de todas as linhas e separados em tokens por vírgula, com `trim` e comparação case-insensitive com `Origin`.
- `Vary: *` também é respeitado: nada é adicionado.
- O valor final junta todas as linhas originais e acrescenta `Origin`, no mesmo formato já usado (separado por `,`).
- Testes novos: várias linhas de `Vary`, `origin` dentro de uma lista, header com "Origin" no nome (`X-Origin-Id`) e `Vary: *`.

Testes:
- `go build ./...` ok.
- `go test ./pkg/middlewares/headers/` ok.
- `golangci-lint run ./pkg/middlewares/headers/...`: 0 issues.
