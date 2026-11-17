# Fix Certificates list filters, navigator.clipboard fallback, and rework returnTo links

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13852](https://github.com/traefik/traefik/pull/13852) |
| Tipo | Correção |
| Área | webui |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | gndz07 |
| Aberto em | 2026-09-07 (última atualização: 2026-10-30) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +168/-37 em 6 arquivos |
| Labels | status/2-needs-review, area/webui, kind/bug/fix, size/M |
| Status | pendente |

## O que faz

Quatro correções na WebUI: filtro da lista de certificados expirados, fallback de cópia quando navigator.clipboard não está disponível (HTTP), limite no parâmetro returnTo dos links e isTruncated nas células da tabela.

## Riscos e situação no upstream

Só WebUI. Exige make generate-webui para reembutir os assets, então não dá para editar à mão. Base v3.7, precisa de adaptação. Status/2-needs-review.

## Principais arquivos

- `webui/src/components/buttons/CopyButton.tsx`
- `webui/src/components/tables/TableFilter.tsx`
- `webui/src/hooks/use-copy-to-clipboard.ts`
- `webui/src/hooks/use-href-with-return-to.spec.tsx`
- `webui/src/hooks/use-href-with-return-to.ts`
- `webui/src/pages/certificates/Certificates.tsx`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13852/head:upstream-pr-13852`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
