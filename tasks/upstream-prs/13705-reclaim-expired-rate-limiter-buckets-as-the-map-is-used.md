# Reclaim expired rate limiter buckets as the map is used

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13705](https://github.com/traefik/traefik/pull/13705) |
| Tipo | Correção |
| Área | middleware/ratelimiter (ttlmap, uso de memória) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | deyanp |
| Aberto em | 2026-08-17 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +93/-5 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, area/middleware |
| Status | importado |

## O que faz

O ttlmap só removia entradas expiradas quando a chave era consultada de novo ou quando o mapa enchia, então cada inMemoryRateLimiter crescia até 65536 entradas e ficava assim. Agora o Set remove entradas expiradas da raiz do heap, limitado a 16 por chamada, sem goroutine de fundo. Resolve #13704.

## Riscos e situação no upstream

Mudança pequena e localizada, com teste. Sem goroutine de fundo, para evitar vazamento a cada reload. Status needs-review, sem review ainda. PR base v3.7.

## Principais arquivos

- `pkg/middlewares/ratelimiter/ttlmap/ttlmap.go`
- `pkg/middlewares/ratelimiter/ttlmap/ttlmap_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13705/head:upstream-pr-13705`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado.**

Achados: correção correta e segura. Hoje uma entrada expirada só sai do `ttlmap` quando a mesma chave é
lida de novo ou quando o mapa enche, então cada limiter in-memory cresce até `maxSources` (65536) e fica
lá. O sweep é limitado a 16 entradas por `Set` (sem stall com o mutex), não cria goroutine (middlewares
não são fechados no reload) e para na primeira entrada viva do heap, então nunca remove entradas vivas.
Conferido o caso em que a chave do próprio `Set` é varrida antes da busca: cai no caminho de inserção, ok.
Nada suspeito no diff.

Relação com o grupo: complementa o #13958 (TTL maior para buckets com burst grande): com TTLs mais longos,
reclamar os expirados de forma contínua mantém a memória proporcional às fontes ativas.

Ajustes: nenhum.

Testes: `go test ./pkg/middlewares/ratelimiter/ttlmap/` ok (três testes novos), `golangci-lint` 0 issues.
