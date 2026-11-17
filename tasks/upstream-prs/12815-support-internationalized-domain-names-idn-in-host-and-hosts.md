# Support internationalized domain names (IDN) in Host and HostSNI matchers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12815](https://github.com/traefik/traefik/pull/12815) |
| Tipo | Recurso |
| Área | muxer/matchers |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | olegnazarov23 |
| Aberto em | 2026-03-15 (última atualização: 2026-11-10) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +131/-19 em 10 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/server |
| Status | pendente |

## O que faz

Os matchers Host e HostSNI convertem domínios com caracteres não-ASCII para punycode via golang.org/x/net/idna, em vez de rejeitá-los. Ex.: Host(`münchen.de`) passa a casar com xn--mnchen-3ya.de. Sem dependência nova, e suporta wildcard.

## Riscos e situação no upstream

Pequeno a médio (10 arquivos). Há questão de escopo em aberto com maintainer: ParseDomains e ParseHostSNI continuam devolvendo a forma unicode, o que afeta ACME e a checagem de HostSNI com TLS terminado.

## Principais arquivos

- `pkg/muxer/http/matcher.go`
- `pkg/muxer/http/matcher_v2.go`
- `pkg/muxer/tcp/matcher.go`
- `pkg/muxer/tcp/matcher_v2.go`
- `pkg/muxer/muxer.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12815/head:upstream-pr-12815`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
