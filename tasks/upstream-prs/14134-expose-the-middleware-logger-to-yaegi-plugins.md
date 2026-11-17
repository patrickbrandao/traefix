# Expose the middleware logger to Yaegi plugins

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14134](https://github.com/traefik/traefik/pull/14134) |
| Tipo | Recurso |
| Área | plugins/yaegi |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | manenim |
| Aberto em | 2026-10-24 (última atualização: 2026-11-15) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +134/-0 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/plugins |
| Status | pendente |

## O que faz

Expõe middlewares.GetLogger para plugins Yaegi, que passam a usar o logger do Traefik com níveis explícitos e campos estruturados. A ponte usa o contexto e o nome do middleware existentes. Captura de stdout/stderr continua igual. Inclui fixture e testes; docs atualizadas.

## Riscos e situação no upstream

Adiciona superfície de API para plugins Yaegi, então qualquer ajuste posterior do contrato vira compatibilidade. Status: needs-review. Base master. Corpo do PR truncado, mas a descrição é clara.

## Principais arquivos

- `pkg/plugins/middlewareyaegi.go`
- `pkg/plugins/middlewareyaegi_test.go`
- `pkg/plugins/fixtures/src/testpluginlogging/testpluginlogging.go`
- `docs/content/reference/install-configuration/experimental/plugins.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14134/head:upstream-pr-14134`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
