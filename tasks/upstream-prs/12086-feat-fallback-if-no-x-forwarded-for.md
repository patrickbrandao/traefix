# Feat fallback if no X-Forwarded-For

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12086](https://github.com/traefik/traefik/pull/12086) |
| Tipo | Correção |
| Área | middleware/ipallowlist, ip strategy |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | clement-brodu |
| Aberto em | 2025-09-20 (última atualização: 2025-10-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +40/-17 em 4 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/S, area/middleware |
| Status | importado com ajustes |

## O que faz

Quando o IPStrategy não encontra IP no X-Forwarded-For, em vez de rejeitar com IP vazio o ipAllowList passa a usar o remoteAddr. Tenta resolver a issue 7884, que quebra acesso local quando excludedIPs está configurado.

## Riscos e situação no upstream

Mexe em decisão de segurança de IP (4 arquivos, +40/-17): cair no remoteAddr pode mudar a confiança em cabeçalhos. Maintainers colocaram status design-review e ainda analisam. Exige cuidado com trusted proxies.

## Atenção: alterações locais do Traefix

Mexe nos forwarded headers, que já foram alterados localmente (193af416c e fd632f41f). Confira se continua compatível.

## Principais arquivos

- `pkg/ip/strategy.go`
- `pkg/ip/strategy_test.go`
- `docs/content/middlewares/http/ipallowlist.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12086/head:upstream-pr-12086`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:
- **Risco de bypass do IPAllowList.** O PR cai incondicionalmente no `RemoteAddr` quando nenhum IP do `X-Forwarded-For` serve. Quando a conexão vem do próprio proxy listado em `excludedIPs` (o caso típico) e esse proxy não manda XFF, ou manda só IPs confiáveis, o IP do proxy vira o IP do cliente. Como o proxy costuma estar dentro do `sourceRange` (mesma LAN), todo tráfego externo que passa por ele seria liberado.
- Também afeta RateLimit, DistributedRateLimit e InFlightReq, que usam a mesma estratégia: todo o tráfego do proxy cairia num único bucket identificado pelo IP do proxy.
- O PR continua em `status/1-needs-design-review`, sem decisão de maintainer. Tem também whitespace sobrando e comentários fora da convenção.
- Não há conflito com os commits locais 193af416c e fd632f41f: eles mexem em `pkg/middlewares/forwardedheaders`, e este PR só em `pkg/ip`. Com o `forwardedHeaders` da entrypoint, o XFF de origem não confiável já chega vazio aqui, que é justamente o caso do acesso direto pela LAN.

Ajustes:
- O fallback para o `RemoteAddr` só acontece quando o próprio `RemoteAddr` **não** está no pool de `excludedIPs`. Se a conexão vem de um proxy confiável sem XFF utilizável, o resultado continua vazio (requisição negada), como antes. A cadeia é tratada como `XFF..., RemoteAddr`, da direita para a esquerda.
- Testes cobrem: XFF todo confiável vindo de remoto confiável (vazio) e de remoto não confiável (IP remoto); XFF ausente com remoto confiável e com remoto não confiável; XFF com IP não confiável tem precedência sobre o remoto.
- Docs: nota nas tabelas de `excludedIPs` de ipallowlist, ratelimit, distributed-ratelimit e inflightreq, e uma seção em `migrate/v3.md`. A documentação antiga (`docs/content/middlewares/http/`) que o PR alterava não existe mais neste branch.

Testes:
- `go build ./...` ok.
- `go test ./pkg/ip/ ./pkg/middlewares/ipallowlist/ ./pkg/middlewares/ipwhitelist/ ./pkg/config/dynamic/` ok.
- `pkg/middlewares/ratelimiter`: os testes in-memory passam. Os `TestRedisRateLimit` falham por tempo ("rate was slower than expected"), o que não tem relação com a mudança: a máquina estava carregada e o código não passa por `ipStrategy`.
- `golangci-lint run ./pkg/ip/...`: 0 issues.
