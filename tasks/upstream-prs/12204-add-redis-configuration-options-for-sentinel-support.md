# Add Redis configuration options for Sentinel support

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12204](https://github.com/traefik/traefik/pull/12204) |
| Tipo | Recurso |
| Área | middleware/ratelimit |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | shkarface |
| Aberto em | 2025-10-28 (última atualização: 2026-07-21) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +48/-13 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware |
| Status | importado com ajustes |

## O que faz

Adiciona suporte a Redis Sentinel para o ratelimit, para ter HA em setups Redis sem cluster. Motivação: uma queda de Redis fez o ratelimit devolver 500. Observação: o provider kv já tem Sentinel (pkg/provider/kv/redis/redis.go), mas o limitador de rate não.

## Riscos e situação no upstream

Pequeno (6 arquivos, +48/-13). Sem testes marcados. Status needs-review, sem comentários de maintainers.

## Principais arquivos

- `pkg/middlewares/ratelimiter/redis_limiter.go`
- `pkg/config/dynamic/middlewares.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12204/head:upstream-pr-12204`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:

- **Vazamento de credencial:** `SentinelUsername` e `SentinelPassword` vinham com `export:"true"`, o que
  faz o redactor/export anonimizado manter o valor. `Username`/`Password` do Redis usam `loggable:"false"`;
  os campos do Sentinel passaram a usar o mesmo tag.
- Comentário de `MasterName` errado ("Default value is the first endpoint"): no go-redis, um `MasterName`
  não vazio é o que liga o modo Sentinel (`FailoverClient`).
- O PR não leva os campos ao CRD do Kubernetes (só file/labels/KV). Mantido assim; a doc agora diz isso.
- Sem testes no PR; o mapeamento é direto para `redis.UniversalOptions`.

Ajustes: tags `loggable:"false"` nas credenciais do Sentinel, comentários corrigidos, doc de `masterName`
explicando o modo Sentinel e a falta de suporte no CRD.

Testes: `go build ./...` ok; `go test ./pkg/config/... ./pkg/redactor/... ./pkg/middlewares/ratelimiter/...` ok
(`TestRedisRateLimit` é sensível a tempo e falhou com a máquina em load ~40; passa isolado);
`golangci-lint run` nos pacotes tocados: 0 issues.
