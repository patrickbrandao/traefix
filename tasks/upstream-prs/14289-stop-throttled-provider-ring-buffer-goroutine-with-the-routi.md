# Stop throttled provider ring buffer goroutine with the routines pool

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14289](https://github.com/traefik/traefik/pull/14289) |
| Tipo | Correção |
| Área | server/configurationwatcher (throttled providers) |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | sylr |
| Aberto em | 2026-11-13 (última atualização: 2026-11-13) |
| Draft | sim |
| Branch-alvo | `master` |
| Tamanho | +50/-5 em 3 arquivos |
| Labels | — |
| Status | pendente |

## O que faz

A goroutine do ringBuffer usado por providers throttled nunca parava, então sobrevivia ao Pool.Stop(). Agora ela roda no pool e termina quando o contexto é cancelado, e o canal de entrada ganha buffer de 100.

## Riscos e situação no upstream

Draft. Vale só para shutdown limpo e testes, sem efeito em operação normal. Depende de #14290 para os testes em synctest. Base é master.

## Principais arquivos

- `ring channel dos providers throttled (newRingChannel)`
- `pool de rotinas (routinespool)`
- `testes de provider throttled`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14289/head:upstream-pr-14289`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
