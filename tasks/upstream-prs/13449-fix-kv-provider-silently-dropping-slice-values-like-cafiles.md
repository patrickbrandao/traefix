# Fix KV provider silently dropping slice values like caFiles when a folder-marker key exists at the same path

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13449](https://github.com/traefik/traefik/pull/13449) |
| Tipo | Correção |
| Área | provider/kv |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | aliamerj |
| Aberto em | 2026-07-01 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +52/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/provider/kv |
| Status | importado com ajustes |

## O que faz

Ignora chaves KV com valor vazio terminadas em '/' (marcador de pasta) antes de agrupar slices. Sem isso, a chave marcador colide com caFiles/0 e sobrescreve o valor real com string vazia, o que quebra o mTLS com 'CAFiles is required'. Fixes #12746.

## Riscos e situação no upstream

PR base v3.7 (portar para master), e um maintainer pediu rebase para v3.6. Sem review. Escopo pequeno (kv_node.go e testes).

## Principais arquivos

- `pkg/config/kv/kv_node.go`
- `pkg/config/kv/kv_node_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13449/head:upstream-pr-13449`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- Bug real: uma chave "pasta" (por exemplo `.../caFiles/` com valor vazio, como a UI do Consul cria) cai em `simplePairs` com a chave `.../caFiles`, depois de normalizada no decode, e colide com o slice `.../caFiles/0..n`. O resultado é que o slice some ou fica vazio, então `caFiles` e listas parecidas são descartadas sem aviso.
- A correção ignora chaves terminadas em `/` com valor vazio antes de agrupar. Uma chave terminada em `/` com valor não vazio continua valendo (há teste).
- Mudança de comportamento mínima: uma pasta vazia isolada não gera mais um nó com valor vazio. Isso não tinha significado na configuração.
- Sem risco de segurança.

**Ajustes:** só um comentário de *porquê* na condição.

**Testes:** `go test ./pkg/config/kv/ ./pkg/provider/kv/...` ok; `go build ./pkg/...` ok; `golangci-lint run ./pkg/config/kv/...` com 0 issues.
