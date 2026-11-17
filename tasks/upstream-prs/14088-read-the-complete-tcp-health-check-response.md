# Read the complete TCP health check response

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14088](https://github.com/traefik/traefik/pull/14088) |
| Tipo | Correção |
| Área | healthcheck/tcp |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | immanuwell |
| Aberto em | 2026-10-18 (última atualização: 2026-10-19) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +62/-1 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/healthcheck |
| Status | importado |

## O que faz

Usa io.ReadFull na resposta do health check TCP. Leituras parciais (ex.: PO e NG em escritas TLS separadas) faziam um backend saudavel ficar DOWN e resetar conexoes roteadas.

## Riscos e situação no upstream

Pequeno (+62/-1, 2 arquivos), mantendo limite de tamanho e deadline. Um review COMMENTED; autor testou TCP, TLS e casos de timeout.

## Principais arquivos

- `pkg/healthcheck/healthcheck_tcp.go`
- `pkg/healthcheck/healthcheck_tcp_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14088/head:upstream-pr-14088`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado.**

Achados no PR original:

- O bug é real. Um único `conn.Read` podia devolver só parte da resposta esperada (por exemplo, `PO` + `NG` em dois registros TLS ou segmentos TCP). Nesse caso o backend saudável era marcado DOWN.
- `io.ReadFull` lê exatamente `len(Expect)` bytes. O tamanho continua limitado por `maxPayloadSize` (64 KiB, validado no construtor) e o deadline de `conn.SetDeadline` continua valendo. Não há leitura sem limite nem risco de travar além do `timeout`. Bytes extras depois do esperado são ignorados, como antes.
- Nada suspeito no diff.

Ajustes: nenhum.

Testes:

- `go test -race ./pkg/healthcheck/`: ok. O teste novo cobre resposta completa, resposta dividida, truncada, vazia e incorreta.
- `golangci-lint run ./pkg/healthcheck/...`: 0 issues.
- `go build ./pkg/...`: ok.
