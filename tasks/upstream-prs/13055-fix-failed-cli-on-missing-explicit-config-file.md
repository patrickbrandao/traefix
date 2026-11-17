# Fix failed cli on missing explicit config file

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13055](https://github.com/traefik/traefik/pull/13055) |
| Tipo | Correção |
| Área | cli/config |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | cyphercodes |
| Aberto em | 2026-04-27 (última atualização: 2026-10-16) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +166/-3 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/server |
| Status | pendente |

## O que faz

Faz --configfile apontando para um arquivo inexistente, ou vazio, retornar erro, em vez de cair silenciosamente nos caminhos padrão. A checagem de deprecação passa a olhar só o arquivo explícito. Resolve o issue #13045.

## Riscos e situação no upstream

Base v3.7. Já tem duas aprovações. Muda comportamento para quem depende do fallback silencioso, e a documentação não foi atualizada.

## Principais arquivos

- `pkg/cli/loader_file.go`
- `pkg/cli/deprecation.go`
- `pkg/cli/loader_file_test.go`
- `pkg/cli/deprecation_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13055/head:upstream-pr-13055`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
