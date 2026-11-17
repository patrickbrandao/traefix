# Allow server healthchecks to run concurrently

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12485](https://github.com/traefik/traefik/pull/12485) |
| Tipo | Recurso |
| Área | healthcheck |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | extndr |
| Aberto em | 2025-12-30 (última atualização: 2026-01-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +131/-35 em 5 arquivos |
| Labels | status/1-needs-design-review, size/M, area/healthcheck |
| Status | pendente |

## O que faz

Executa as healthchecks dos servers de um service em paralelo, reduzindo a latência de monitoramento quando há muitos backends. Nova opção maxConcurrency, com 1 restaurando o comportamento anterior. Fecha a issue #9748.

## Riscos e situação no upstream

Muda o comportamento padrão de healthcheck (agora paralelo). Status design-review. Toca healthcheck.go e a config HTTP.

## Principais arquivos

- `pkg/healthcheck/healthcheck.go`
- `pkg/healthcheck/healthcheck_test.go`
- `pkg/config/dynamic/http_config.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12485/head:upstream-pr-12485`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
