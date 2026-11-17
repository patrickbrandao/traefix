# Support TCP middleware plugin

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12264](https://github.com/traefik/traefik/pull/12264) |
| Tipo | Recurso |
| Área | tcp/plugins |
| Prioridade | **P4** |
| Valor | baixa |
| Esforço de port | alto |
| Autor | david-garcia-garcia |
| Aberto em | 2025-11-12 (última atualização: 2026-07-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +1339/-103 em 43 arquivos |
| Labels | status/1-needs-design-review, size/L, area/tcp, area/plugins |
| Status | pendente |

## O que faz

Permite que plugins Yaegi e WASM implementem middleware TCP via NewTCP(), e muda o Serve de plugins TCP para receber um context, para suportar metadados e tempo de vida por conexão.

## Riscos e situação no upstream

43 arquivos (+1339/-103) e mudança de assinatura interna em plugins TCP. Status design-review, sem aprovação. Só vale se o operador usar plugins.

## Principais arquivos

- `pkg/plugins/middlewareyaegi.go`
- `pkg/plugins/middlewarewasm.go`
- `pkg/plugins/builder.go`
- `pkg/config/dynamic/tcp_middlewares.go`
- `pkg/middlewares/tcp/`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12264/head:upstream-pr-12264`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
