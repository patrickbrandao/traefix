# Do not reuse the consumed request body for mirrors when mirrorBody is false

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13316](https://github.com/traefik/traefik/pull/13316) |
| Tipo | Correção |
| Área | middleware/mirroring |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | JATINKUMAR1008 |
| Aberto em | 2026-06-06 (última atualização: 2026-09-23) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +77/-3 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | importado com ajustes |

## O que faz

Com mirrorBody: false, os clones espelhados passam a ter http.NoBody, ContentLength zero, TransferEncoding limpo e sem Content-Length, em vez de reusar o body já consumido e fechado pelo handler principal. Corrige 502 falso no mirror ('invalid Read on closed Body') que polui métricas e traces. Fixes #13292.

## Riscos e situação no upstream

PR base v3.7 (portar para master). Review APPROVED. Testes de regressão para body de tamanho conhecido e chunked. Escopo pequeno.

## Principais arquivos

- `pkg/server/service/loadbalancer/mirror/mirror.go`
- `pkg/server/service/loadbalancer/mirror/mirror_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13316/head:upstream-pr-13316`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes** (adaptação ao código atual).

Análise:
- O bug procede. Com `mirrorBody: false`, os clones espelhados compartilhavam o `Body` da requisição original e mantinham `Content-Length`/`TransferEncoding`. Depois que o handler principal consome e o servidor fecha o corpo, o espelho falha com `http: invalid Read on closed Body`, e isso gerava 502 nas métricas e nos traces do espelho. Também havia uma leitura concorrente do mesmo `Body` entre o handler principal e a goroutine do espelho.
- O fix é mínimo e correto: `http.NoBody`, `ContentLength = 0`, `TransferEncoding = nil` e remoção do header `Content-Length`, só nos clones do espelho. A requisição principal não é alterada.
- Aprovado por um maintainer (amazon7737). Nenhum problema de segurança. O fix também elimina uma leitura concorrente do corpo.

Ajustes:
- O patch não aplicava direto. O código local já extraiu `mirrorContext`, e `New`/`AddMirror` têm mais parâmetros. Fix e testes foram reaplicados à mão.
- No teste chunked, a flag `mirrored` virou `atomic.Bool`: ela é escrita na goroutine do pool e lida no teste.
- Comentário do fix reescrito para explicar o porquê (o corpo pode ter sido fechado pelo servidor).

Testes:
- `go test -race -count=1 ./pkg/server/service/loadbalancer/mirror/` ok.
- `golangci-lint run ./pkg/server/service/loadbalancer/mirror/...`: 0 issues.
- `go build ./...` ok.
