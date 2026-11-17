# Keep existing Set-Cookie headers when adding custom response headers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14105](https://github.com/traefik/traefik/pull/14105) |
| Tipo | Correção |
| Área | middleware/headers |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-10-20 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +50/-3 em 3 arquivos |
| Labels | status/0-needs-triage, kind/bug/fix, size/S, area/middleware |
| Status | importado com ajustes |

## O que faz

O middleware headers, ao aplicar customResponseHeaders, usa Header.Set e apaga os Set-Cookie já existentes. Com sticky sessions e um headers middleware que define cookie, o cookie de afinidade se perde. A correção acrescenta Set-Cookie em vez de substituir; os demais headers continuam sendo sobrescritos.

## Riscos e situação no upstream

Mudança pequena e localizada, com testes e docs atualizados. Escopo restrito a Set-Cookie. Status: aguardando triagem dos maintainers (status/0-needs-triage). Base v3.7, cherry-pick deve ser simples.

## Principais arquivos

- `pkg/middlewares/headers/header.go`
- `pkg/middlewares/headers/header_test.go`
- `docs/content/reference/routing-configuration/http/middlewares/headers.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14105/head:upstream-pr-14105`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes** (só resolução de conflito).

Análise:
- O bug procede (#14103). O cookie de sticky session é adicionado à resposta antes do proxy, e um `Set-Cookie` em `customResponseHeaders` fazia `Header.Set`, que apagava todos os `Set-Cookie` existentes, inclusive o de afinidade.
- O fix passa a usar `Add` só para `Set-Cookie`. Pela RFC 6265, cada cookie vai num header `Set-Cookie` separado, então acrescentar é seguro. Os demais headers continuam sobrescritos, e valor vazio continua removendo o header.
- Mudança de comportamento: quem usava `customResponseHeaders` para *substituir* um `Set-Cookie` do backend agora envia os dois, e o navegador aplica o último para o mesmo nome. A doc foi atualizada. Para remover os do backend, continua valendo `Set-Cookie: ""`, e agora também dá para reescrevê-los com `replaceResponseHeadersRegex` (#12563).
- Interação com o #12563: a substituição por regex roda depois dos custom headers, então também atua sobre o cookie acrescentado. Comportamento coerente.
- Segurança: nenhum problema. O cookie de sessão deixa de ser apagado sem aviso.

Ajustes:
- Conflito no fim de `header_test.go` com os testes do #12563, resolvido mantendo os dois.

Testes:
- `go test -count=1 ./pkg/middlewares/headers/` ok.
- `golangci-lint run ./pkg/middlewares/headers/...`: 0 issues.
