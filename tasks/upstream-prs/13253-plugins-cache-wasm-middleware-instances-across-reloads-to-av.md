# plugins: cache wasm middleware instances across reloads to avoid runtime leaks

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13253](https://github.com/traefik/traefik/pull/13253) |
| Tipo | Correção |
| Área | plugins/wasm |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | Yaminyam |
| Aberto em | 2026-05-29 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +323/-36 em 2 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/plugins |
| Status | pendente |

## O que faz

Mantém em cache (por nome do middleware + hash da config do guest) o middleware http-wasm, o runtime wazero e o módulo compilado, com contagem de referências. Recargas da config dinâmica deixam de criar um runtime novo a cada vez, e runtimes substituídos são fechados de forma determinística. Ataca vazamento de runtime em plugins WASM (parcial, relacionado a #13235).

## Riscos e situação no upstream

Muda o ciclo de vida de runtimes WASM (refcount, evicção, finalizers): risco de concorrência e de fechar runtime em uso. Status 1-needs-design-review, sem review; descrição tem inconsistência de versão alvo (v3.6 vs v3.7) e 'Closes nothing'. Comentários pedindo atenção há meses.

## Principais arquivos

- `pkg/plugins/middlewarewasm.go`
- `pkg/plugins/middlewarewasm_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13253/head:upstream-pr-13253`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
