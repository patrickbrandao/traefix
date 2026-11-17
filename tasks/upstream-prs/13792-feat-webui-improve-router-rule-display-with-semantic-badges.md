# feat(webui): improve router rule display with semantic badges

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13792](https://github.com/traefik/traefik/pull/13792) |
| Tipo | Recurso |
| Área | webui |
| Prioridade | **P4** |
| Valor | baixa |
| Esforço de port | alto |
| Autor | xsetra |
| Aberto em | 2026-08-27 (última atualização: 2026-10-27) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +604/-51 em 13 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, area/webui, size/L |
| Status | pendente |

## O que faz

Na WebUI, as regras de router viram badges coloridos por tipo de matcher (Host, Path, Method, Header etc.), com operadores visuais e 'mais N' para listas longas. Ajuda a diagnosticar regras complexas com o mesmo host e PathPrefixes diferentes (issues #11987 e #11378).

## Riscos e situação no upstream

Só UI, 604 linhas em 13 arquivos. Status status/1-needs-design-review (design-review aberto pelo maintainer). Autor pediu merge há dois meses e reportou conflito. Requer make generate-webui para reembutir os assets, o que não pode ser feito à mão.

## Principais arquivos

- `webui/src/components/resources/DetailItemComponents.tsx`
- `webui/src/components/resources/DetailsCard.tsx`
- `webui/src/components/routers/RouterFlowDiagram.tsx`
- `webui/src/components/routers/RuleDisplay.tsx`
- `webui/src/libs/rule-parser.spec.ts`
- `webui/src/libs/rule-parser.ts`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13792/head:upstream-pr-13792`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
