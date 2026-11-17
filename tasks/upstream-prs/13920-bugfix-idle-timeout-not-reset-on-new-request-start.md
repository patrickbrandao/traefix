# bugfix: idle-timeout not reset on new request start

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13920](https://github.com/traefik/traefik/pull/13920) |
| Tipo | Correção |
| Área | service/idle-timeout |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | smirco-zoox |
| Aberto em | 2026-09-16 (última atualização: 2026-11-02) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +68/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/service |
| Status | pendente |

## O que faz

Corrige timer de idle que expirava no meio de uma requisição em conexões com baixa taxa de requisições, gerando 504 espúrios quando o intervalo entre requests é quase igual ao idle timeout. O timer passa a ser zerado no início de cada nova requisição.

## Riscos e situação no upstream

Pequena, com testes; descrição indica uso de IA na autoria. Status needs-review, sem review ainda; documentação não atualizada. Toca transport.go.

## Principais arquivos

- `pkg/server/service/transport.go`
- `pkg/server/service/transport_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13920/head:upstream-pr-13920`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
