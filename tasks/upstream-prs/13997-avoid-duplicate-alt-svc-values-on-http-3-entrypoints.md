# Avoid duplicate Alt-Svc values on HTTP/3 entryPoints

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13997](https://github.com/traefik/traefik/pull/13997) |
| Tipo | Correção |
| Área | entrypoint/http3 |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | minseojo |
| Aberto em | 2026-10-01 (última atualização: 2026-11-01) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +119/-1 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/http3 |
| Status | pendente |

## O que faz

Quando HTTP/3 esta ativo, o Traefik preserva o proprio valor de Alt-Svc mesmo que backends adicionem outros valores depois. Evita anunciar uma porta HTTP/3 inexistente no Traefik (caso h3=":9443" vindo do backend, #12082).

## Riscos e situação no upstream

Pequeno (+119/-1) e reutiliza middlewares.NewResponseModifier. Label status/1-needs-design-review. Primeiro PR de contribuidor novo: CI aguardando aprovacao de maintainer.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp_http3.go`
- `pkg/server/server_entrypoint_tcp_http3_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13997/head:upstream-pr-13997`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
