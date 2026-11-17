# Wait for initial TLS configuration before accepting traffic

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14219](https://github.com/traefik/traefik/pull/14219) |
| Tipo | Correção |
| Área | server/startup, tls, acme |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | escooterclinic |
| Aberto em | 2026-11-03 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +686/-34 em 11 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, area/acme, size/L, area/tls, area/server |
| Status | pendente |

## O que faz

Antes de aceitar conexões, o Traefik espera pelo primeiro snapshot dos providers e pela aplicação bem-sucedida de TLS e roteamento. Assim, uma conexão que chega cedo não recebe o certificado padrão gerado quando já existe certificado válido local. Erros de TLS na inicialização abortam o startup. Reloads continuam best-effort.

## Riscos e situação no upstream

Muda a semântica de startup e de listeners (+686/-34, 11 arquivos). Status: needs-design-review; é resubmissão de outro PR (#14218) via fork pessoal. Alto risco de regressão no boot, inclusive com providers lentos. Há testes de startup.

## Principais arquivos

- `pkg/server/server.go`
- `pkg/server/configurationwatcher.go`
- `cmd/traefik/traefik.go`
- `pkg/provider/acme/startup_test.go`
- `pkg/server/server_startup_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14219/head:upstream-pr-14219`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
