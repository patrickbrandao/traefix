# Keep labeled containers when inspect fails on port ranges

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13576](https://github.com/traefik/traefik/pull/13576) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | young0264 |
| Aberto em | 2026-07-27 (última atualização: 2026-07-30) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +77/-1 em 2 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/M |
| Status | adiado |

## O que faz

Quando o inspect do container não pode ser decodificado (ex.: port range após a migração para os módulos moby), inspectContainers devolvia dados vazios e o container era descartado. Agora o container é reconstruído a partir do resumo de ContainerList já obtido. Resolve #13146.

## Riscos e situação no upstream

Reconstrução a partir de dados resumidos, então alguns campos ficam ausentes nesse caminho. Status needs-review, sem review de maintainer ainda. PR base v3.7.

## Principais arquivos

- `pkg/provider/docker/pdocker.go`
- `pkg/provider/docker/shared_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13576/head:upstream-pr-13576`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- O issue de origem (upstream 13146) foi reportado no Traefik 2.11, que usa o cliente Docker antigo. O PR não mostra que a falha de decode do `inspect` com `expose` em faixa (`12000-13000`) ainda acontece com o cliente `moby/moby/client` e a `moby/moby/api` v1.56 usados hoje, e o teste simula um erro genérico (`assert.AnError`), não o erro real de decode.
- **A correção é ineficaz:** o `dockerData` de fallback leva só ID, nome e labels, sem `NetworkSettings`. Assim `getIPAddress` não acha IP, `addServer` falha e o container é descartado do mesmo jeito, só que um passo depois e com outra mensagem de erro.
- **Contorna o filtro de estado:** o fallback não copia `State`/`Status` do summary. Com `Status == ""`, `keepContainer` trata o container como "em execução". Um container parado (ou removido entre o `list` e o `inspect`, caso NotFound, também coberto pelo fallback) entraria no pipeline como se estivesse ativo.
- O fallback vale para qualquer erro de `inspect`, inclusive timeouts e erros de permissão, não só para o erro de decode que se queria tratar.

**Veredito:** adiado. Para avançar é preciso primeiro reproduzir o erro com o cliente atual. Se ainda ocorrer, o fallback deve se limitar ao erro de decode e montar o `dockerData` a partir do summary completo (`State`, `NetworkSettings.Networks`, `Ports`, `HostConfig.NetworkMode`), ou o decode da faixa deve ser corrigido na origem.

**Testes:** não aplicável (nada importado).
