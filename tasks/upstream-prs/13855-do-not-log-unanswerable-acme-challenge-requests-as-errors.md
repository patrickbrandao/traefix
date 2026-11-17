# Do not log unanswerable ACME challenge requests as errors

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13855](https://github.com/traefik/traefik/pull/13855) |
| Tipo | Correção |
| Área | acme/logs |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | datrixlab |
| Aberto em | 2026-09-07 (última atualização: 2026-09-10) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +77/-3 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/acme, size/M |
| Status | pendente |

## O que faz

Rebaixa para debug três linhas de ERROR no handler de HTTP-01 challenge (path malformado e token inexistente) e adiciona o primeiro arquivo de teste do handler. Os 404 já estavam corretos; só a severidade muda.

## Riscos e situação no upstream

Mudança pequena de nível de log, mas o log some do nível padrão de operação. Label kind/enhancement e status needs-review, sem review de maintainer registrado.

## Principais arquivos

- `pkg/provider/acme/challenge_http.go`
- `pkg/provider/acme/challenge_http_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13855/head:upstream-pr-13855`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
