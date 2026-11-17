# fix(middlewares): share redis rate-limiter clients across instances

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13619](https://github.com/traefik/traefik/pull/13619) |
| Tipo | Correção |
| Área | middleware/ratelimiter (backend Redis, vazamento de goroutines) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | funkypenguin |
| Aberto em | 2026-08-01 (última atualização: 2026-08-20) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +168/-1 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware, ai/review |
| Status | importado com ajustes |

## O que faz

Cada reload da configuração criava um novo cliente go-redis por instância do rate limiter, e os antigos nunca eram fechados, o que faz goroutines e memória crescerem sem parar. A proposta é um cache de clientes indexado pela config Redis e um fingerprint dos materiais TLS, para que certificados rotacionados gerem cliente novo. Resolve #13618.

## Riscos e situação no upstream

Alternativa à #13676 (que fecha clientes em vez de compartilhá-los). Os dois PRs tocam o mesmo arquivo e não devem ser portados juntos sem decidir. Status review, com review COMMENTED.

## Principais arquivos

- `pkg/middlewares/ratelimiter/redis_limiter.go`
- `pkg/middlewares/ratelimiter/redis_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13619/head:upstream-pr-13619`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes (combinado com o #13676).**

Achados no PR original:

- O leak é real: um `redis.NewUniversalClient` por instância de middleware, por router, a cada reload,
  sem `Close`; desde o go-redis v9.12 cada cliente mantém uma goroutine viva, então nunca é coletado.
- A chave do cache era o JSON da configuração em texto puro, ou seja, senha do Redis e, com TLS inline,
  a chave privada ficavam duplicadas como chave de map pelo tempo de vida do processo.
- **Clientes substituídos nunca eram fechados** (apontado também pelo review automático do upstream):
  cada mudança de configuração do Redis ou rotação de certificado continuava vazando um cliente.
- O fingerprint TLS fazia hash do valor literal quando a leitura falhava; isso é redundante, pois o
  valor literal já está no JSON.
- Testes dependiam de contagem do map global (`before+1`) e de mensagens longas nos asserts.

Ajustes:

- Chave do cache = SHA-256 do JSON da configuração + conteúdo dos arquivos TLS (CA/cert/key). Nenhuma
  credencial fica em texto na chave.
- Comentários enxutos, só o porquê.
- Testes reescritos (`assert.Same`/`NotSame`, sem contar o map global, verificação de que a chave não
  contém a senha, rotação do CA no mesmo caminho muda a chave).
- O fechamento dos clientes sem uso vem no commit do #13676, que agora conta referências sobre este cache
  (ver a task do #13676). Os dois PRs se sobrepõem: o #13619 sozinho não fecha nada, e o #13676 sozinho
  recria e fecha um pool por router a cada reload. Combinados, há um cliente por configuração distinta, e
  ele é fechado quando o último limiter que o usa é coletado.

Sobreposição com o #13823 (adiado): aquele PR também reduzia o número de pools Redis compartilhando o
limiter no `Builder`; com este cache isso já está resolvido para o Redis.

Testes: `go test ./pkg/middlewares/ratelimiter/` (novos `TestRedisLimiter_sharedClientPerConfig`,
`TestRedisClientKey`) ok; `go build ./...` ok; `golangci-lint` 0 issues.
