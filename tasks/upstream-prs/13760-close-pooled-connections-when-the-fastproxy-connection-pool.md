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
| Status | pendente |

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
