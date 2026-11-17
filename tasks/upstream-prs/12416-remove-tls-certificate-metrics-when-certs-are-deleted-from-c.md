# Remove tls certificate metrics when certs are deleted from config

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12416](https://github.com/traefik/traefik/pull/12416) |
| Tipo | Correção |
| Área | observability/metrics/tls |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | Takalele |
| Aberto em | 2025-12-13 (última atualização: 2025-12-18) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +11/-1 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware/metrics |
| Status | importado com ajustes |

## O que faz

Ao recarregar a configuração, zera as métricas de certificados TLS antes de recriar as atuais. Assim certificados removidos do dynamic config deixam de aparecer no Prometheus sem precisar reiniciar o Traefik. Corrige a issue #12358.

## Riscos e situação no upstream

Pequeno (2 arquivos, +11/-1). Só triagem, sem review de maintainer. Toca cmd/traefik/traefik.go e o registro do Prometheus.

## Principais arquivos

- `cmd/traefik/traefik.go`
- `pkg/observability/metrics/prometheus.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12416/head:upstream-pr-12416`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- A mudança é pequena e correta: a cada atualização da configuração dinâmica, o GaugeVec
  `traefik_tls_certs_not_after` do Prometheus é zerado antes de ser repopulado. Assim, certificados removidos
  deixam de ser reportados.
- Existe uma janela curta entre o `Reset` e a repopulação em que um scrape pode não ver nenhum certificado.
  É aceitável, porque no scrape seguinte as séries voltam.
- A correção cobre só o Prometheus. Datadog, StatsD, InfluxDB2 e OTel continuam sem limpar séries antigas, mas
  nesses backends o gauge é push e a série simplesmente para de ser enviada.
- Não há nada suspeito nem dependência nova.

Ajustes:
- O comentário em `cmd/traefik/traefik.go` agora explica o porquê e termina em ponto.
- Adicionei o doc comment da função exportada `ResetTLSCertificateMetrics`.
- `prometheusState.reset()` (helper de teste) também limpa o novo campo `tlsCertsGaugeVec`.
- Adicionei o teste `TestResetTLSCertificateMetrics`.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/observability/metrics/`: ok.
- `golangci-lint run ./pkg/observability/metrics/... ./cmd/traefik/...`: 0 issues.
