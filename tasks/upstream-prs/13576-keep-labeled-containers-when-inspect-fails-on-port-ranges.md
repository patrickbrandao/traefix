# Keep labeled containers when inspect fails on port ranges

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13576](https://github.com/traefik/traefik/pull/13576) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | young0264 |
| Aberto em | 2026-07-27 (última atualização: 2026-07-30) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +77/-1 em 2 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/M |
| Status | pendente |

## O que faz

Quando o inspect do container não pode ser decodificado (ex.: port range após a migração para os módulos moby), inspectContainers devolvia dados vazios e o container era descartado. Agora o container é reconstruído a partir do resumo de ContainerList já obtido. Resolve #13146.

## Riscos e situação no upstream

Reconstrução a partir de dados resumidos, então alguns campos ficam ausentes nesse caminho. Status needs-review, sem review de maintainer ainda. PR base v3.7.

## Principais arquivos

- `pkg/provider/docker/pdocker.go`
- `pkg/provider/docker/shared_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13576/head:upstream-pr-13576`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
