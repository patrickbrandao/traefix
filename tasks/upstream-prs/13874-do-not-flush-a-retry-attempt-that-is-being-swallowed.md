# Do not flush a retry attempt that is being swallowed

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13874](https://github.com/traefik/traefik/pull/13874) |
| Tipo | Correção |
| Área | middleware/retry |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | datrixlab |
| Aberto em | 2026-09-09 (última atualização: 2026-09-14) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +120/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware |
| Status | pendente |

## O que faz

Corrige o Flush do responseWriter do retry, que não verifica shouldNotWrite nem written. Flushar uma tentativa descartada faz o net/http enviar um 200 implícito ao cliente enquanto o loop de retry ainda roda, o que impede o failover. Alinha com o padrão já usado em customerrors.

## Riscos e situação no upstream

Mudança de uma função de guarda com teste de regressão. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/middlewares/retry/retry.go`
- `pkg/middlewares/retry/retry_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13874/head:upstream-pr-13874`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
