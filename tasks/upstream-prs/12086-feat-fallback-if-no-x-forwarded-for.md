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
| Status | pendente |

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
