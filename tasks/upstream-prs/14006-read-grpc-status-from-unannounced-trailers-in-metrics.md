# Read gRPC status from unannounced trailers in metrics

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14006](https://github.com/traefik/traefik/pull/14006) |
| Tipo | Correção |
| Área | middleware/metrics |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-10-04 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +21/-3 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware/metrics |
| Status | importado |

## O que faz

A metrica de gRPC le Grpc-Status tambem de http.TrailerPrefix quando o header simples esta vazio. Sem isso, backends como Kestrel que enviam trailers nao anunciados registram todas as chamadas bem-sucedidas com code="2" (Unknown).

## Riscos e situação no upstream

Pequeno (+21/-3), em um arquivo de metrics. Um review COMMENTED; autor pediu revisao de maintainer em comentario recente.

## Principais arquivos

- `pkg/middlewares/metrics/metrics.go`
- `pkg/middlewares/metrics/metrics_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14006/head:upstream-pr-14006`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Correção pequena e correta (fixes #14005): quando o backend envia `grpc-status` como trailer sem anunciá-lo (Kestrel), o reverse proxy grava `Trailer:Grpc-Status` no header, e as métricas registravam toda chamada bem-sucedida como `code="2"` (Unknown).
- `Header().Get(http.TrailerPrefix + "Grpc-Status")` funciona porque a chave com `:` não é canonicalizada, e o proxy grava exatamente esse formato.
- Sem risco de cardinalidade: o valor passa por `codes.Code.UnmarshalJSON`; valores inválidos viram `Unknown`.
- Só review automatizada; sem maintainer humano.

### Ajustes

Nenhum: aplicado como está (com o teste do PR).

### Testes

- `go test ./pkg/middlewares/metrics/`: ok.
- `golangci-lint run ./pkg/middlewares/metrics/...`: 0 issues.
