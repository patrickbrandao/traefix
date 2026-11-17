# Make duplicate slash normalization optional

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12879](https://github.com/traefik/traefik/pull/12879) |
| Tipo | Recurso |
| Área | server/entrypoints/sanitizePath |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | boinger |
| Aberto em | 2026-03-26 (última atualização: 2026-08-10) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +447/-6 em 10 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/server |
| Status | pendente |

## O que faz

Adiciona a opção de entrypoint mergeSlashes (padrão true), que controla a junção de barras consecutivas em sanitizePath. Com false, // é preservado e segmentos . e .. continuam sendo resolvidos. Corrige a issue #12666.

## Riscos e situação no upstream

Mexe na sanitização de path, que é superfície de segurança (path traversal, autorização de gateways a montante). Precisa de review de segurança. Status review sem maintainer comentando.

## Principais arquivos

- `pkg/config/static/entrypoints.go`
- `pkg/server/server_entrypoint_tcp.go`
- `docs/content/reference/install-configuration/entrypoints.md`
- `docs/content/security/request-path.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12879/head:upstream-pr-12879`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
