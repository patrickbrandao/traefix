# Add ifAcceptContains option to errors middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13967](https://github.com/traefik/traefik/pull/13967) |
| Tipo | Recurso |
| Área | middleware/errors |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | FenTiger |
| Aberto em | 2026-09-26 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +70/-2 em 4 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/S, area/middleware |
| Status | pendente |

## O que faz

Adiciona ao middleware errors a opção ifAcceptContains, para aplicar páginas de erro customizadas só a clientes que pedem um certo tipo no header Accept (por exemplo text/html, de navegadores). APIs continuam recebendo a resposta original.

## Riscos e situação no upstream

Maintainers colocaram status design-review. Autor relata uso em homelab por meses. Docs marcadas pelo autor como possivelmente incorretas.

## Principais arquivos

- `pkg/middlewares/customerrors/custom_errors.go`
- `pkg/middlewares/customerrors/custom_errors_test.go`
- `pkg/config/dynamic/middlewares.go`
- `docs/content/reference/routing-configuration/http/middlewares/errorpages.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13967/head:upstream-pr-13967`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
