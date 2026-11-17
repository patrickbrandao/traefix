# Do not expire rate limiter buckets before they refill

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13958](https://github.com/traefik/traefik/pull/13958) |
| Tipo | Correção |
| Área | middleware/ratelimit |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | rasoanaivo-r |
| Aberto em | 2026-09-24 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +87/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware |
| Status | importado com ajustes |

## O que faz

Corrige o TTL dos buckets do rate limiter, que expirava em cerca de 1/rate e recriava o bucket cheio. Com burst alto e rate baixo, pausas curtas davam rajadas completas repetidas. O TTL passa a ser max(ttl atual, ceil(burst/rate)), e o Redis herda a mesma correção.

## Riscos e situação no upstream

Aumenta a vida dos buckets (memória limitada por maxSources). Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13958/head:upstream-pr-13958`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:

- Bug de segurança real (bypass do rate limit): o TTL do bucket dependia só da taxa, e um bucket expirado
  volta cheio. Com taxa baixa e burst grande (cotas por hora), um cliente que pausa por alguns segundos
  recebe o burst inteiro de novo; o relato no PR mostra ~60x o limite configurado. Afeta os dois backends,
  porque o mesmo TTL vai para o `EXPIRE` do Redis.
- A correção `ttl = max(ttl, ceil(burst / rate))` está certa: um bucket só expira quando já estaria cheio.
  Com `average: 0` (`rate.Inf`) o refill é 0 e o TTL não muda.
- Faltava limite: com burst enorme e taxa minúscula, `int(math.Ceil(...))` pode estourar (conversão de
  float fora do range é indefinida em Go) e `time.Duration(ttl) * time.Second` no `ttlmap` também.

Ajustes: TTL limitado a `math.MaxInt32` segundos (constante `maxTTL`), com caso de teste
(`Burst: math.MaxInt64`, `Average: 1` por 24 h). Código reescrito com `min`/`max`.

Relação com o grupo: buckets vivem mais tempo; o #13705 (sweep do `ttlmap`) evita que expirados se
acumulem, e a memória continua limitada por `maxSources`.

Testes: `go test ./pkg/middlewares/ratelimiter/` (`TestNewRateLimiter_ttl` com o caso novo) ok,
`golangci-lint` 0 issues.
