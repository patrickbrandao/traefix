# feat(middleware): add generic OR-chain support (resolves #12249)

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12393](https://github.com/traefik/traefik/pull/12393) |
| Tipo | Recurso |
| Área | middleware/chain |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | wchsiang |
| Aberto em | 2025-12-09 (última atualização: 2025-12-11) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +77/-5 em 2 arquivos |
| Labels | status/0-needs-triage, size/M |
| Status | rejeitado |

## O que faz

Introduz o rótulo opcional orGroup nos middlewares: middlewares com o mesmo grupo são avaliados em OR, e grupos diferentes são combinados em AND. Sem orGroup o comportamento atual é mantido. Resolve o caso de ipAllowList em OR sem duplicar faixas de IP.

## Riscos e situação no upstream

Mexe na semântica da cadeia de middlewares (núcleo). Maintainers colocaram em espera: caso de uso incerto, janela de 6 meses para ver tração na issue antes de revisar. Sem docs nem testes além dos do próprio PR.

## Principais arquivos

- `pkg/config/dynamic/middlewares.go`
- `pkg/server/middleware/middlewares.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12393/head:upstream-pr-12393`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: rejeitado.**

Achados no PR original:
- **DoS de CPU e memória.** `newORChainHandler` chama o construtor de cada middleware do grupo a **cada requisição**, com parse de config, criação de checker, logger e wrappers de observabilidade. Erros de construção são ignorados em silêncio (`continue`).
- Usa `httptest.NewRecorder()` em código de produção. A resposta do middleware é descartada, mas o handler pode ter consumido o corpo da requisição ou causado efeitos colaterais (ex.: ForwardAuth chamado N vezes, contadores de rate limit incrementados).
- **Muda a ordem da cadeia.** Os grupos OR são postos *antes* de todos os middlewares AND, em ordem alfabética do nome do grupo, o que quebra a regra de que a cadeia segue a ordem declarada no router. Um middleware de auth declarado primeiro pode passar a rodar depois de um headers/strip, e vice-versa.
- A semântica "chamou `next` = autorizou" só serve para middlewares de filtro. Para os demais (headers, redirect, compress) o resultado não faz sentido e o efeito do middleware se perde.
- Lê `midInf.OrGroup` de `runtime.MiddlewareInfo` sem tags TOML/labels. Não tem testes nem docs, e tem formatação quebrada (tabs/espaços, sem newline no fim do arquivo).
- O maintainer (nmengin) deixou a proposta em `need-triage` por 6 meses, aguardando tração da issue #12249.

Decisão: rejeitado pelos problemas de desenho e de segurança acima. O traefik/traefik#13509 resolve o mesmo caso de uso (#12249) de forma restrita ao IPAllowList e sem esses problemas, mas também ficou adiado (ver a task do #13509).

Testes: nenhum, porque o PR não foi importado.
