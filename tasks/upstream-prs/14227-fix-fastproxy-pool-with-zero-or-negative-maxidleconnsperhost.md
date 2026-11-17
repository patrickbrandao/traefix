# Fix FastProxy pool with zero or negative maxIdleConnsPerHost

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14227](https://github.com/traefik/traefik/pull/14227) |
| Tipo | Correção |
| Área | proxy/fastproxy |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | hood |
| Aberto em | 2026-11-04 (última atualização: 2026-11-11) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +137/-3 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/fastproxy |
| Status | pendente |

## O que faz

O pool de conexões do FastProxy passa a seguir a doc para maxIdleConnsPerHost: 0 usa o default do net/http (2), e valores negativos não reutilizam conexões. Hoje o negativo causa panic (makechan: size out of range) e o zero pode travar no acquire.

## Riscos e situação no upstream

Escopo pequeno (connpool.go). Só afeta quem usa FastProxy com esse valor. Status: needs-review, base v3.7. Docs já descreviam o comportamento.

## Principais arquivos

- `pkg/proxy/fast/connpool.go`
- `pkg/proxy/fast/connpool_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14227/head:upstream-pr-14227`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
