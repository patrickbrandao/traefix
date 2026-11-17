# Add poolSize and maxActiveConns configuration to the Redis KV provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14052](https://github.com/traefik/traefik/pull/14052) |
| Tipo | Recurso |
| Área | provider/kv/redis |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | stackedbyaradhya |
| Aberto em | 2026-10-12 (última atualização: 2026-11-11) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +16/-8 em 3 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/provider/redis |
| Status | pendente |

## O que faz

Expoe poolSize e maxActiveConns no provider KV Redis (providers.redis), que ja existem no cliente Redis do kvtools mas nao eram repassados. Permite ajustar o pool de conexoes da descoberta de configuracao. Resolve #14051.

## Riscos e situação no upstream

Pequeno (+16/-8). Sem testes automatizados, por limitacao do cliente de teste; autor perguntou por andamento. Um review COMMENTED.

## Principais arquivos

- `pkg/provider/kv/redis/redis.go`
- `docs/content/reference/install-configuration/providers/kv/redis.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14052/head:upstream-pr-14052`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
