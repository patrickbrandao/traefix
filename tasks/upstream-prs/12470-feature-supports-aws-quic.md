# feature: Supports AWS QUIC

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12470](https://github.com/traefik/traefik/pull/12470) |
| Tipo | Recurso |
| Área | server/http3 |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | kyounghoonJang |
| Aberto em | 2025-12-26 (última atualização: 2026-04-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +201/-1 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/server |
| Status | pendente |

## O que faz

Adiciona configuração de QUIC Server ID no HTTP/3, injetado por variável de ambiente (AWS_LBC_QUIC_SERVER_ID), para que o Traefik funcione atrás de um NLB da AWS com listeners QUIC. Fecha a issue #12425.

## Riscos e situação no upstream

Útil só para quem usa NLB com QUIC na AWS. Comentário de revisão aponta que a variável é base64 e não hex, então a decodificação do PR pode estar errada. Sem review de maintainer.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp_http3.go`
- `pkg/server/server_entrypoint_tcp_http3_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12470/head:upstream-pr-12470`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
