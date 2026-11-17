# FastCGI protocol support

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11732](https://github.com/traefik/traefik/pull/11732) |
| Tipo | Recurso |
| Área | transport/fastcgi |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | nvlad613 |
| Aberto em | 2025-05-04 (última atualização: 2026-04-30) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +1316/-0 em 15 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L |
| Status | pendente |

## O que faz

Adiciona um transporte FastCGI (RoundTripper) para o Traefik proxiar requisições para backends como PHP-FPM, com root, splitPathRegex, timeouts, mapeamento de env e pool de conexões. O comportamento foi modelado no do nginx.

## Riscos e situação no upstream

Código novo grande (15 arquivos, +1316 linhas) com pool de conexões próprio. O autor reconhece que é projeto de capstone e não marcou testes. Status needs-design-review, sem reviews. Risco de manutenção alto.

## Principais arquivos

- `pkg/server/service/fastcgi/transport.go`
- `pkg/server/service/fastcgi/client.go`
- `pkg/server/service/fastcgi/conn.go`
- `pkg/server/service/transport.go`
- `pkg/config/dynamic/http_config.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11732/head:upstream-pr-11732`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
