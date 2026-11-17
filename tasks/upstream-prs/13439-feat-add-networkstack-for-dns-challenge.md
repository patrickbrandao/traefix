# feat: add networkStack for DNS challenge

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13439](https://github.com/traefik/traefik/pull/13439) |
| Tipo | Recurso |
| Área | acme/dns-challenge |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | ldez |
| Aberto em | 2026-06-30 (última atualização: 2026-11-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +56/-6 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/acme, size/S |
| Status | pendente |

## O que faz

Adiciona uma opção para definir a pilha de rede (network stack) usada pelo DNS challenge do ACME. Fixes #13302, segundo a descrição.

## Riscos e situação no upstream

Reviewer questionou o vínculo com #12446, e a referência foi removida. Reviews só COMMENTED. Sem testes (item marcado como não aplicado). Escopo pequeno, em um arquivo de provider e docs.

## Principais arquivos

- `pkg/provider/acme/provider.go`
- `docs/content/reference/install-configuration/configuration-options.md`
- `docs/content/https/ref-acme.toml`
- `docs/content/https/ref-acme.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13439/head:upstream-pr-13439`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
