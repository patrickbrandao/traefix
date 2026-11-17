# fix: instantiate a fresh WASI host per request for wasm middlewares

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13617](https://github.com/traefik/traefik/pull/13617) |
| Tipo | Correção |
| Área | plugins/wasm (WASI) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | gterminator |
| Aberto em | 2026-07-31 (última atualização: 2026-08-03) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +261/-24 em 10 arquivos |
| Labels | status/1-needs-design-review, size/L, area/plugins |
| Status | pendente |

## O que faz

Um único wasi.System (dono dos file descriptors reais) era compartilhado por todas as requisições de um middleware wasm. Com concorrência, uma requisição podia ler bytes de outra socket ou colidir em fd (Bad file number), o que explica o sintoma NOT_LEADER_FOR_PARTITION da issue #11629. O fix cria um WASI por requisição.

## Riscos e situação no upstream

Vazamento de dados entre requisições, o que dá peso alto. Mexe em go.mod e num fixture com plugin.wasm binário. Status needs-design-review, com maintainers pedindo análise de impacto. Sem review ainda.

## Principais arquivos

- `pkg/plugins/wasip.go`
- `pkg/plugins/wasip_mock.go`
- `pkg/plugins/middlewarewasm.go`
- `pkg/plugins/middlewarewasm_test.go`
- `pkg/plugins/middlewarewasm_socket_test.go`
- `go.mod`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13617/head:upstream-pr-13617`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
