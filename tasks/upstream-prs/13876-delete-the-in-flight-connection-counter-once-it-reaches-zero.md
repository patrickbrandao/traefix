# Delete the in-flight connection counter once it reaches zero

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13876](https://github.com/traefik/traefik/pull/13876) |
| Tipo | Correção |
| Área | middleware/tcp-inflightconn |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | datrixlab |
| Aberto em | 2026-09-09 (última atualização: 2026-09-22) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +41/-6 em 2 arquivos |
| Labels | status/2-needs-review, contributor/waiting-for-corrections, kind/bug/fix, size/S, area/middleware |
| Status | importado com ajustes |

## O que faz

O mapa de conexões do middleware inFlightConn (TCP) guarda uma entrada zerada por IP de origem para sempre. A correção remove a chave quando o contador chega a zero, o que alinha com o comportamento do HTTP inFlightReq e não muda o resultado, já que increment trata chave ausente como zero.

## Riscos e situação no upstream

Pequeno, com correção de race no teste existente. Review do maintainer pediu rebase para v3.7 (é bugfix), o que aqui deve ser lido como ajuste de base; o fork está em master. Status contributor/waiting-for-corrections.

## Principais arquivos

- `pkg/middlewares/tcp/inflightconn/inflight_conn.go`
- `pkg/middlewares/tcp/inflightconn/inflight_conn_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13876/head:upstream-pr-13876`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados: o mapa `connections` do middleware TCP `inFlightConn` guardava uma entrada zerada por IP de
origem para sempre (crescimento sem limite, DoS lento de memória num entrypoint exposto). O `delete`
quando o contador chega a zero é seguro: `increment` lê chave ausente como zero, e tudo ocorre sob o mesmo
mutex. Mesmo padrão do `connlimit` do oxy usado no HTTP. Nada suspeito no diff.

Ajustes: o código local já tinha a correção de sincronização de `TestInFlightConn_ServeTCP` (fecha
`finishCh` depois que `ServeTCP` retorna); a versão do PR desse teste conflitou e foi descartada em favor
da local. Mantidos o teste novo `TestInFlightConn_ReleasedConnectionsAreForgotten` e o `fakeConn.Close`
tolerante a canal nil.

Testes: `go test -count=50 ./pkg/middlewares/tcp/inflightconn/` ok, `golangci-lint` 0 issues.
