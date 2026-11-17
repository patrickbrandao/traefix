# Stop mutating shared provider precedence slices in place

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13761](https://github.com/traefik/traefik/pull/13761) |
| Tipo | Correção |
| Área | server, config/static |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | dgprivate |
| Aberto em | 2026-08-25 (última atualização: 2026-08-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +68/-3 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/server |
| Status | pendente |

## O que faz

SetDefaults e SetEffectiveConfiguration escrevem em slices compartilhados (providerNames), o que gera data race detectada por go test -race. Hoje o efeito é nulo porque todos os nomes já estão em minúsculas, mas deixa de ser verdade se um provider com maiúscula for adicionado.

## Riscos e situação no upstream

Base v3.7. Mudança pequena e segura. Sem efeito de runtime hoje. Status/2-needs-review, review não verificado.

## Principais arquivos

- `pkg/config/static/providers_defaults_test.go`
- `pkg/config/static/static_config.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13761/head:upstream-pr-13761`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
