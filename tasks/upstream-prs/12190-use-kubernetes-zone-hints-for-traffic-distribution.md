# Use kubernetes zone hints for traffic distribution

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12190](https://github.com/traefik/traefik/pull/12190) |
| Tipo | Recurso |
| Área | provider/kubernetes |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | sylr |
| Aberto em | 2025-10-23 (última atualização: 2026-04-13) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +233/-10 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/M, area/provider/k8s/crd, area/provider/k8s |
| Status | rejeitado |

## O que faz

Usa as zone hints do Kubernetes (traffic distribution) para preferir endpoints da mesma zona nos providers Ingress, CRD e Gateway. Visa reduzir tráfego entre zonas.

## Riscos e situação no upstream

Toca os três providers Kubernetes (6 arquivos, +233/-10). Sem testes nem docs marcados. Autor usa variante em produção com pesos 0/100 e pede opinião dos maintainers sobre o default. Só 'COMMENTED' em review.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/crd/kubernetes_http.go`
- `pkg/provider/kubernetes/gateway/kubernetes.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12190/head:upstream-pr-12190`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: rejeitado.**

Achados:
- Peso 0 para endpoints de outras zonas: o WRR nunca manda tráfego a eles, então se os endpoints da zona local ficarem indisponíveis (health check, falha) não há fallback e o serviço responde 503. O autor admite que a versão que usa em produção é diferente.
- No provider Gateway API o informer de Nodes passa a ser criado incondicionalmente, sem opção de desligar: instalações sem RBAC de `nodes` travam em `WaitForCacheSync`.
- Configuração implícita por variáveis de ambiente (`KUBERNETES_NODE_NAME`/`NODE_NAME`) lidas a cada build de serviço, e varredura de todos os nós por serviço.
- Nenhum teste nem documentação; lógica triplicada nos três providers; o PR começou como draft e ainda pede opinião dos maintainers sobre o default.

Motivo: risco de indisponibilidade (sem fallback entre zonas) e regressão de RBAC; abordagem sem consenso no upstream.

Testes: nenhum (não importado).
