# Fix duplicate logging fields [#192]

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13038](https://github.com/traefik/traefik/pull/13038) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress, logging |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | pvbouwel |
| Aberto em | 2026-04-24 (última atualização: 2026-08-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +13/-5 em 2 arquivos |
| Labels | status/0-needs-triage, area/provider/k8s/ingress, size/S, area/provider/k8s |
| Status | rejeitado |

## O que faz

Evita mutar o logger obtido do contexto ao registrar "Skipping service: no endpoints found". Hoje cada chamada acrescenta campos namespace/ingress, o que faz as linhas de log crescerem sem limite e até ficarem truncadas (JSON inválido) em clusters grandes.

## Riscos e situação no upstream

Base v3.7 e label need-triage. O autor não conseguiu escrever um teste de regressão e a correção foi validada só em cluster próprio. Toca os providers ingress e ingress-nginx.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13038/head:upstream-pr-13038`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados**

- O PR alega que o logger do contexto é "mutado" e acumula campos `namespace`/`ingress` sem limite. No código atual não há nenhum `UpdateContext` nos providers Kubernetes Ingress/ingress-nginx; cada iteração cria um logger novo com `log.Ctx(ctx).With()`, que copia o contexto do zerolog e não altera o original. Não consegui reproduzir o acúmulo, e o próprio autor diz que não conseguiu escrever um teste que o reproduza.
- A correção troca o contexto recebido por `context.Background()` e um logger global com `"kubernetes"` fixo no código. Isso vai contra o AGENTS.md (propagar o `ctx` do chamador) e descarta campos e o logger configurado no contexto.
- O trecho do ingress-nginx não se aplica mais: o provider foi bastante reestruturado (`build.go`).

**Decisão**

Rejeitado. Falta reprodução, a abordagem piora a propagação de contexto e o bug não aparece no código atual. Se o problema voltar a acontecer, o melhor é investigar onde o logger é guardado e reaproveitado (por exemplo, em `routerTransform` ou em plugins), em vez de cortar o contexto.

**Testes**

Nenhum (nada importado).
