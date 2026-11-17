# Add denyOnError option to the Redis rate limiter

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13916](https://github.com/traefik/traefik/pull/13916) |
| Tipo | Recurso |
| Área | middleware/ratelimit-redis |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | deyanp |
| Aberto em | 2026-09-16 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +221/-10 em 16 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware |
| Status | importado com ajustes |

## O que faz

Adiciona a opção denyOnError ao rate limiter com Redis. Com true (padrão, como hoje) erros do Redis geram 500; com false, o erro é logado e a requisição passa sem limite. Evita que uma falha do store derrube toda a API quando o rate limit serve para conter abuso.

## Riscos e situação no upstream

Retoma o PR #13529 (commits do autor original preservados). Corrige default no provider REST, que não chama SetDefaults. Toca 16 arquivos, incluindo CRDs gerados (exige make generate-crd). Status needs-review, sem review ainda.

## Principais arquivos

- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `docs/content/reference/routing-configuration/http/middlewares/ratelimit.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13916/head:upstream-pr-13916`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.** Escolhido no lugar do #13529 (que ele contém) e do #12042.

Achados:

- O PR corrige os dois problemas do #13529: `DenyOnError` virou `*bool` (o provider REST não chama
  `SetDefaults`, e com `bool` um bloco `redis` sem a opção passava a fail-open) e o in-memory continua
  fail-closed sem bloco `redis`. A conversão no CRD copia o valor (não aponta para o objeto do cache do
  informer). Default seguro mantido: `true`.
- O review automático do upstream acusou `new(true)` como erro de compilação; é falso positivo (Go ≥ 1.26
  aceita `new(valor)`; o `go.mod` local é 1.27).
- Risco inerente, documentado agora: com `denyOnError: false`, quem conseguir derrubar ou saturar o Redis
  desliga o rate limit. É opt-in.
- Sobreposição com #12042 (adiado): aquele propõe circuit breaker; aqui cada requisição ainda paga o
  timeout do Redis durante a queda (padrão 3 s de leitura / 5 s de dial). Fica como evolução possível.
- Sobreposição com #12204 (Sentinel) e #13619 (cache de clientes): conflito de texto em doc e struct,
  resolvido mantendo os dois. `denyOnError` entra no JSON da chave do cache de clientes, então
  configurações que só diferem nele usam clientes distintos: inofensivo.

Ajustes:

- No fail-open, o log diz explicitamente que a requisição seguiu sem rate limiting.
- Doc de `redis.denyOnError` alerta que desligar a opção permite contornar o limite derrubando o Redis.
- Código gerado conferido com `make generate-crd` (o último passo do script usa `sed -i` GNU e falha no
  macOS; rodado manualmente com `sed -i ''`) e `go generate`: saída idêntica ao PR, árvore limpa.

Testes: `go build ./...` ok; `go test` de `pkg/middlewares/ratelimiter` (incluindo
`TestRateLimiterDenyOnError` e os casos novos de `TestNewRateLimiter`), `pkg/provider/kubernetes/crd`,
`pkg/config/dynamic`, `pkg/redactor` ok; `golangci-lint` 0 issues.
