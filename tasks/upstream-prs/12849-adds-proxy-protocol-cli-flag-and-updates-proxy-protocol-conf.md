# Adds proxy protocol cli flag and updates proxy protocol config docs

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12849](https://github.com/traefik/traefik/pull/12849) |
| Tipo | Recurso |
| Área | server/proxyprotocol, config/static |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | mismithhisler |
| Aberto em | 2026-03-19 (última atualização: 2026-09-21) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +195/-3 em 7 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/server |
| Status | pendente |

## O que faz

Permite configurar via CLI a versão padrão do proxyProtocol em tcpServersTransport (default@internal). Também atualiza a documentação dos serversTransport com os padrões reais que o Traefik aplica.

## Riscos e situação no upstream

Pequeno (7 arquivos). Aprovado por maintainer, que considerou a falta um bug. Pediram rebase para a branch v3.6 (maintenance), o que o fork precisa adaptar.

## Principais arquivos

- `pkg/config/static/static_config.go`
- `pkg/provider/traefik/internal.go`
- `pkg/tcp/dialer.go`
- `docs/content/reference/install-configuration/configuration-options.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12849/head:upstream-pr-12849`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
