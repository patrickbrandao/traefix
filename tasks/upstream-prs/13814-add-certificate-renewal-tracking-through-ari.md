# Add certificate renewal tracking through ARI

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13814](https://github.com/traefik/traefik/pull/13814) |
| Tipo | Recurso |
| Área | acme, tls |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | amarkdotdev |
| Aberto em | 2026-08-31 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +246/-10 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/acme, size/L |
| Status | importado com ajustes |

## O que faz

Renova certificados ACME quando o servidor sugere via ACME Renewal Information (RFC 9773). Útil para revogações em massa ou mudanças de CA. A renovação por tempo continua valendo; o ARI só antecipa. Honra Retry-After quando for menor que o intervalo.

## Riscos e situação no upstream

Depende da lib lego com o suporte de Renew/UseARICertID (go-acme/lego#3273), então pode exigir bump de dependência. Servidores sem ARI mantêm o comportamento atual. Status/2-needs-review, review não verificado. Fecha #12461 e #13808.

## Principais arquivos

- `pkg/provider/acme/provider.go`
- `pkg/provider/acme/provider_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13814/head:upstream-pr-13814`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados (PR original, aprovado por jnoordsij; versão reescrita por amazon7737 sobre o lego v5.6.0):
- Implementa ARI (RFC 9773): para cada certificado que ainda não está no período de renovação por tempo, o
  provider busca a renewal info e renova quando a janela sugerida já começou. Isso só pode antecipar uma
  renovação, nunca adiá-la. A ordem de renovação usa `UseARICertID` (`replaces`). O `Retry-After` encurta a
  próxima checagem, sem passar do `renewInterval`. Servidores sem ARI retornam `api.ErrNoARI` sem fazer
  request. Não há dependência nova: tudo existe no lego v5.6.0 já usado.
- O snapshot de `p.certificates` é clonado e o lock é liberado antes das chamadas de rede. Isso está correto
  e evita segurar o RWMutex durante I/O.
- **`getClient()` era chamado uma vez por certificado** dentro de `checkRenewalInfo`. Quando o client não pode ser
  criado (CA fora do ar, falha de registro da conta), `p.client` continua nil e cada certificado refazia
  `initAccount` e a criação do client, multiplicando timeouts em cada checagem. Isso inclui a checagem
  síncrona do `Provide()`.
- **O piso de `Retry-After` era 1 minuto.** Se o CA sugere renovação e ela falha (rate limit, erro de
  desafio), o próximo ciclo pode ocorrer em 1 minuto e tentar de novo todas as renovações sugeridas. Isso
  martela o CA e consome o rate limit da conta.
- Os testes usam só `httptest` local e não há chamada de rede externa.

Ajustes:
- O client ACME é obtido uma única vez por ciclo de renovação, de forma preguiçosa, só se houver certificado
  para checar. Se falhar, a checagem ARI do ciclo é pulada (log em debug). `checkRenewalInfo` virou função
  pura, que recebe o `*lego.Client`.
- `minRenewalInfoRetryAfter` subiu de 1 minuto para 1 hora, com o comentário explicando o porquê (rate limit
  em falhas repetidas). O piso nunca adia uma checagem além do `renewInterval`, já que vale
  `min(renewInterval, max(retryAfter, piso))`. Por isso certificados de vida curta continuam com o intervalo
  deles.
- Testes ajustados para a nova assinatura (`Test_checkRenewalInfo`). O conflito com o teste do #13439 foi
  resolvido mantendo os dois.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/provider/acme/`: ok.
- `golangci-lint run ./pkg/provider/acme/...`: 0 issues.
