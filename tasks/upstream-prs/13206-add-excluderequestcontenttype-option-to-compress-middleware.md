# Add ExcludeRequestContentType option to compress middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13206](https://github.com/traefik/traefik/pull/13206) |
| Tipo | Recurso |
| Área | middlewares/compress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | KaanSimsek |
| Aberto em | 2026-05-19 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +106/-42 em 8 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware |
| Status | importado com ajustes |

## O que faz

Adiciona a opção excludeRequestContentType ao middleware compress. Com ela desligada, o Content-Type da requisição deixa de decidir se a resposta será comprimida, e só o Content-Type da resposta é avaliado. Fecha o issue #12070, por exemplo para text/event-stream.

## Riscos e situação no upstream

Comentário positivo de maintainer, sem aprovação formal. Toca o redactor (testdata) e o label, então precisa checar o valor padrão para não mudar o comportamento atual.

## Principais arquivos

- `pkg/middlewares/compress/compress.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/middlewares/compress/compress_test.go`
- `docs/content/reference/routing-configuration/http/middlewares/compress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13206/head:upstream-pr-13206`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.** Escolhido no lugar do traefik/traefik#12066 e aplicado sobre o #14235.

Análise:
- O PR cria a opção `excludeRequestContentType` (padrão `true`). Com `false`, o `Content-Type` da *requisição* deixa de ser comparado a `excludedContentTypes`, e só o da resposta conta (issues #12014 e #12070). O comportamento padrão não muda, ao contrário do #12066.
- O maintainer (nmengin) elogiou a abordagem.
- Achados:
  - **Regressão silenciosa de default.** O campo era `bool` com `SetDefaults() = true`. Providers que não aplicam defaults (o REST provider decodifica JSON direto, com `json.NewDecoder`) passariam a ter `false`, ou seja, mudança de comportamento sem ninguém pedir.
  - Sem `omitempty`, o campo aparecia em toda exportação JSON/TOML/YAML. Por isso o PR teve de alterar os testdata do redactor e os testes de labels e de KV.
  - **Sem suporte no CRD Kubernetes**: `v1alpha1.Compress` não ganhou o campo, então a opção não podia ser desligada via CRD.
  - O comentário de porquê ("Notably for text/event-stream ... #2576") foi removido.
  - Nenhum problema de segurança: `application/grpc` continua excluído pelo `Content-Type` da resposta.

Ajustes:
- O campo virou `*bool` com `omitempty`. `nil` equivale a `true` (`ptr.Deref(..., true)`), no mesmo padrão de `mirrorBody`. Assim nenhum provider muda de comportamento, e o `SetDefaults` e os testes de labels, KV e redactor ficam como estavam.
- Campo adicionado ao CRD (`v1alpha1.Compress.ExcludeRequestContentType`) e à conversão em `createCompressMiddleware`. Deepcopy, applyconfiguration e CRDs foram regenerados com `script/code-gen.sh`.
- Restaurado o comentário de porquê sobre `text/event-stream`.
- Exemplos de referência `file.{toml,yaml}` atualizados.
- Testes:
  - os casos existentes voltaram a usar o padrão (`nil`);
  - os casos novos usam `new(false)`;
  - novo `TestCreateCompressMiddleware` no provider CRD (nil, defaults, opção desligada).

Testes:
- `go build ./...` ok.
- `go test ./pkg/middlewares/compress/ ./pkg/provider/kubernetes/crd/ ./pkg/config/... ./pkg/redactor/ ./pkg/provider/kv/` ok.
- `golangci-lint run ./pkg/middlewares/compress/... ./pkg/provider/kubernetes/crd/... ./pkg/config/dynamic/...`: 0 issues.
