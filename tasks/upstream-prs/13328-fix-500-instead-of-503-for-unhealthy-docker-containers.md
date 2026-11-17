# Fix 500 instead of 503 for unhealthy Docker containers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13328](https://github.com/traefik/traefik/pull/13328) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | sakiphan |
| Aberto em | 2026-06-08 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +243/-18 em 2 arquivos |
| Labels | status/1-needs-design-review, area/provider/docker, kind/bug/fix, size/L |
| Status | pendente |

## O que faz

Com allowEmptyServices e servidor definido por labels, um container unhealthy (ou não running com allowNonRunning) passa a retornar 503, e não 500. Os servers vindos das labels são descartados quando o load balancer é mantido vazio. Aplicado aos builders HTTP, TCP e UDP. Regressão desde a v3.7.1 (issue #13261).

## Riscos e situação no upstream

PR base v3.7 (portar para master). Sem review nem comentários. Status 1-needs-design-review. Mexe nos três builders de serviço, então precisa de cobertura nos três.

## Principais arquivos

- `pkg/provider/docker/config.go`
- `pkg/provider/docker/config_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13328/head:upstream-pr-13328`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
