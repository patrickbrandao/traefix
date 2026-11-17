# [enhancement] Persist HTTP/TCP/UDP table search and status filters across page navigation

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13669](https://github.com/traefik/traefik/pull/13669) |
| Tipo | Recurso |
| Área | webui (tabelas de recursos) |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | Esdro |
| Aberto em | 2026-08-10 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +158/-4 em 2 arquivos |
| Labels | kind/enhancement, area/webui, size/L, bot/light-review |
| Status | pendente |

## O que faz

Os filtros de busca e status das tabelas do dashboard (HTTP Routers, Services, TCP Middlewares etc.) se perdem ao navegar ou recarregar. Este PR mantém esses filtros entre páginas.

## Riscos e situação no upstream

Muda o WebUI, que precisa ser regenerado com make generate-webui para entrar no binário. Status review, sem review ainda. Conveniência de UI, sem efeito no roteamento.

## Principais arquivos

- `webui/src/components/tables/TableFilter.tsx`
- `webui/src/components/tables/TableFilter.spec.tsx`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13669/head:upstream-pr-13669`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
