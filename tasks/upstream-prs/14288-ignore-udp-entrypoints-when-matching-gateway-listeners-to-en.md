# Ignore UDP entryPoints when matching Gateway listeners to entryPoints

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14288](https://github.com/traefik/traefik/pull/14288) |
| Tipo | Correção |
| Área | provider/kubernetes-gateway |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | Neffi42 |
| Aberto em | 2026-11-13 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +118/-1 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/gatewayapi |
| Status | importado com ajustes |

## O que faz

O provider Gateway casava listeners com entrypoints só pela porta, então um entrypoint UDP na mesma porta de um TCP podia ser escolhido por ordem de iteração do mapa. Agora os entrypoints UDP são ignorados nesse casamento, e um listener que só bate com UDP vira PortUnavailable.

## Riscos e situação no upstream

Base é v3.7, precisa de port. Relevante para setups com HTTP/3 que usam a mesma porta em TCP e UDP (443). Mudança de status: listener antes aceito passa a PortUnavailable. Sem review de maintainer ainda (status/2-needs-review).

## Principais arquivos

- `pkg/provider/kubernetes/gateway (casamento listener x entrypoint)`
- `pkg/provider/kubernetes/gateway (testes)`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14288/head:upstream-pr-14288`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Bug real: `entryPointName` casava listeners e entry points só pelo sufixo da porta, e `GetAddress()` remove o `/udp`. Com HTTP/3 (443 em TCP e UDP), a escolha dependia da ordem aleatória de iteração do mapa: um listener HTTPS/TLS podia cair no entry point UDP e ficar sem tráfego, de forma intermitente entre rebuilds.
- Conferido que o provider Gateway API do fork não suporta UDPRoute nem listeners UDP, então ignorar entry points UDP não remove nenhuma funcionalidade.
- Mudança de status: um listener que só casava com um entry point UDP passa a `PortUnavailable`, o que é o correto.
- Teste de TCPRoute do PR em conflito com o fork (nomes de routers diferentes); a expectativa do caso HTTP não batia (o fork gera `TLS.Options` vazio).

Ajustes:
- Caso de TCPRoute conflitante substituído por `TestProvider_entryPointName_ignoresUDP`, que repete a busca para cobrir a aleatoriedade do mapa (TCP+UDP na mesma porta e só UDP).
- Expectativa do caso HTTP ajustada ao fork.

Testes: `go build ./...` ok; `go test ./pkg/provider/kubernetes/gateway/ ./pkg/config/static/` ok; `golangci-lint` nos dois pacotes: 0 issues.
