# feat: enhance headers middleware to support regex replacement

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12563](https://github.com/traefik/traefik/pull/12563) |
| Tipo | Recurso |
| Área | middleware/headers |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jayl1e |
| Aberto em | 2026-01-17 (última atualização: 2026-04-13) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +308/-2 em 9 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware, area/provider/k8s |
| Status | importado com ajustes |

## O que faz

Adiciona ao middleware headers a substituição de headers por regex, sem dependências novas. Um header só é substituído se existir, sem criar um novo, o que resolve o caso da issue #5809 (por exemplo, reescrever Location em redirects).

## Riscos e situação no upstream

Médio (9 arquivos, +308/-2, inclui CRD e regeneração). Comentários de usuários e maintainer confirmaram a semântica de substituir sem adicionar. Sem review de maintainer ainda.

## Principais arquivos

- `pkg/middlewares/headers/header.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/config/dynamic/zz_generated.deepcopy.go`
- `docs/content/reference/routing-configuration/other-providers/file.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12563/head:upstream-pr-12563`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:
- **Bug de precedência que podia causar panic:** `return h != nil && (A || B) || C || D`. Com `h == nil`, a avaliação continua em `len(h.Replace...)` e desreferencia o ponteiro nulo. Hoje os chamadores passam um valor não nulo, mas o método existe justamente para tratar `nil`.
- **Perda de valores em headers múltiplos:** usava `Header.Get` + `Header.Set`, ou seja, só o primeiro valor era considerado e todos os outros eram apagados. Num `Set-Cookie` com vários cookies, uma substituição apagava os demais.
- Um `regex` vazio era aceito. Ele casa entre todos os caracteres e corromperia o valor inteiro.
- A mensagem de erro copiada ("origin parsing") não dizia qual header estava com regex inválida. `Replacement` passava por `TrimSpace`, o que impedia substituições intencionais com espaço.
- Faltava doc de referência do middleware (só os YAML de CRD gerados e os `file.*` foram alterados). Os testes de resposta cobriam só `Foo`.
- Nenhuma dependência nova. As regex são RE2 (tempo linear), então não há risco de ReDoS. O valor da substituição vem da config do operador e não aceita CR/LF de terceiros, porque headers recebidos não podem conter CR/LF.
- Upstream: em "review", sem objeção de maintainer.

Ajustes:
- Corrigida a precedência em `HasCustomHeadersDefined`.
- A substituição é aplicada a **cada valor** do header, e os valores múltiplos são preservados. Um valor que fica vazio é removido, e o header é removido se não sobrar nenhum valor. Um header ausente nunca é criado.
- O nome do header é canonicalizado na construção, então `location` funciona como `Location`.
- Regex vazia ou inválida gera erro na criação do middleware, com o nome do header e da opção.
- A substituição roda depois de `customRequestHeaders`/`customResponseHeaders` (ordem documentada).
- Deepcopy e CRDs regenerados com `script/code-gen.sh` (`make generate-crd`). No macOS o `sed -i` final do script falha (sintaxe BSD), então esse passo foi aplicado manualmente com `sed -i ''`.
- Docs: linhas na tabela de `headers.md` e exemplos em `reference/dynamic-configuration/file.{toml,yaml}`.
- Testes: substituição simples, nome não canônico, sem match, header ausente, vários valores, valor removido, header removido, interação com custom headers, `Location`, vários `Set-Cookie`, regex inválida e regex vazia.

Testes:
- `go build ./...` ok.
- `go test ./pkg/middlewares/headers/ ./pkg/config/dynamic/ ./pkg/config/label/ ./pkg/redactor/` ok.
- Teste ad-hoc (não commitado) confirmou que labels como `traefik.http.middlewares.m.headers.replaceresponseheadersregex.Location.regex` são decodificados.
- `golangci-lint run ./pkg/middlewares/headers/... ./pkg/config/dynamic/...`: 0 issues.
