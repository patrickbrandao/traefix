# Close pooled connections when the FastProxy connection pool is closed

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13760](https://github.com/traefik/traefik/pull/13760) |
| Tipo | Correção |
| Área | fastproxy |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | dgprivate |
| Aberto em | 2026-08-25 (última atualização: 2026-08-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +145/-1 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/fastproxy |
| Status | importado com ajustes |

## O que faz

connPool.Close() não drena as conexões ociosas. Como ProxyBuilder.Update() fecha e descarta o pool a cada mudança de ServersTransport, as conexões ficam vazando (socket, goroutine readLoop e bufio de 64 KiB). Passa a drenar sempre.

## Riscos e situação no upstream

Base v3.7, precisa de adaptação. Pequeno e com teste que falha no branch atual. Afeta só o FastProxy experimental. Status/2-needs-review, review não verificado.

## Principais arquivos

- `pkg/proxy/fast/connpool.go`
- `pkg/proxy/fast/connpool_close_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13760/head:upstream-pr-13760`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, só com review automatizado (traefik-agent), sem maintainer.

### Achados no PR original

- Correto no essencial: `connPool.Close()` não fechava as conexões ociosas. A cada mudança de `ServersTransport` o
  `ProxyBuilder` descarta o pool, e as conexões (com a goroutine `readLoop` e um buffer de 64 KiB cada) ficavam abertas
  até o backend fechá-las. É um vazamento sem limite em clusters com configuração dinâmica que muda com frequência.
- **Possível travamento de request:** o `AcquireConn` espera no canal `idleConns` pela conexão que acabou de discar (o
  `askForNewConn` a entrega pelo canal). O dreno do PR tira e descarta o que estiver no canal: se o `Close` rodar
  enquanto um request em andamento no proxy antigo espera essa conexão, ela é roubada e o request fica bloqueado para
  sempre. O código local do Traefix já tem um `closeIdleConns` (pools dedicados NTLM/Kerberos) que devolve as conexões
  fechadas ao canal justamente para acordar esse waiter.
- **Vazamento residual:** conexões em uso quando o pool é fechado voltam pelo `ReleaseConn` depois do `Close` e ficavam
  presas num pool inacessível.
- `Close` não era idempotente: uma segunda chamada fazia `close(doneCh)` de novo e entrava em pânico.

### Ajustes

- `Close` usa o `closeIdleConns` local, que fecha as conexões e as devolve marcadas como quebradas: um waiter recebe uma
  conexão obsoleta e disca de novo, em vez de travar.
- Flag atômica `closed`: `Close` passa a ser idempotente (`CompareAndSwap`). O `ReleaseConn` fecha direto as conexões
  devolvidas depois do `Close` e verifica de novo depois de devolver, para cobrir a corrida com o `Close`. Esse
  tratamento fica só no `ReleaseConn`: no caminho de discagem ele causaria um loop entre fechar e rediscar.
- Teste novo `TestConnPoolReleaseAfterClose` (conexão devolvida depois do `Close`, e `Close` chamado duas vezes).

### Testes

- `go test -race ./pkg/proxy/fast/` ok; `golangci-lint run --new-from-rev=HEAD ./pkg/proxy/fast/...`: 0 issues.
