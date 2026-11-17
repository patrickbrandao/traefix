# Fix(tracing): use X-Forwarded-For for client.address when trusted

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13444](https://github.com/traefik/traefik/pull/13444) |
| Tipo | Correção |
| Área | middleware/tracing |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | aliamerj |
| Aberto em | 2026-06-30 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +83/-6 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware/tracing |
| Status | pendente |

## O que faz

O atributo de span client.address passa a usar o X-Forwarded-For quando ele vem de peer confiável. network.peer.address continua com o IP TCP real. Não cria lógica nova de confiança, porque forwardedHeaders já remove o header de peers não confiáveis. Fixes #13408.

## Riscos e situação no upstream

PR base v3.7; maintainer pediu rebase para v3.6/v3.7. Sem review. Escopo pequeno (2 arquivos). Valor restrito a quem usa tracing e trustedIPs.

## Principais arquivos

- `pkg/observability/tracing/tracing.go`
- `pkg/middlewares/observability/entrypoint_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13444/head:upstream-pr-13444`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
