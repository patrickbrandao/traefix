# Expose Allow0RTT and InitialPacketSize as HTTP3 configuration options.

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13187](https://github.com/traefik/traefik/pull/13187) |
| Tipo | Recurso |
| Área | entrypoints/http3 |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | davefinster |
| Aberto em | 2026-05-17 (última atualização: 2026-05-18) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +91/-9 em 7 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/server, area/http3 |
| Status | pendente |

## O que faz

Expõe como opções de entrypoint HTTP/3 o Allow0RTT e o InitialPacketSize do QUIC. O InitialPacketSize ajuda em redes com problemas de MTU, como o caso de Tailscale. Hoje o Allow0RTT está fixo em false no código do fork.

## Riscos e situação no upstream

Allow0RTT tem implicação de segurança (replay de 0-RTT), e um maintainer já pediu análise nesse ponto. Autor adicionou documentação na segunda commit. Diff pequeno (+91/-9) em server_entrypoint_tcp_http3.go e entrypoints.go.

## Principais arquivos

- `pkg/config/static/entrypoints.go`
- `pkg/server/server_entrypoint_tcp_http3.go`
- `pkg/server/server_entrypoint_tcp_http3_test.go`
- `docs/content/reference/install-configuration/entrypoints.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13187/head:upstream-pr-13187`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
