# Add RFC 7239 Forwarded header support

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12725](https://github.com/traefik/traefik/pull/12725) |
| Tipo | Recurso |
| Área | middleware/forwardedheaders |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | sammiee5311 |
| Aberto em | 2026-02-23 (última atualização: 2026-10-02) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +112/-0 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware |
| Status | pendente |

## O que faz

Adiciona ao middleware forwardedheaders a remoção do header Forwarded de clientes não confiáveis e a geração de um Forwarded RFC 7239 com IP, protocolo e host do cliente. Fecha a issue #4099.

## Riscos e situação no upstream

Sobrepõe-se a commits locais do fork: 193af416c (Add Forwarded and X-Real-Port headers) e fd632f41f (Remove Forwarded header from untrusted sources) já fazem a mesma sanitização e geração. Importar este PR causaria conflito. Review só com COMMENTED.

## Atenção: alterações locais do Traefix

Sobrepõe os commits locais 193af416c (Forwarded e X-Real-Port) e fd632f41f (Forwarded de origem não confiável). Compare antes: o Traefix já gera e sanitiza o Forwarded, e a mudança local tem prioridade.

## Principais arquivos

- `pkg/middlewares/forwardedheaders/forwarded_header.go`
- `pkg/middlewares/forwardedheaders/forwarded_header_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12725/head:upstream-pr-12725`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
