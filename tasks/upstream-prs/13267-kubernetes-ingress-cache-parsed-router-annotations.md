# kubernetes ingress: cache parsed router annotations

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13267](https://github.com/traefik/traefik/pull/13267) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | ishioni |
| Aberto em | 2026-06-02 (última atualização: 2026-06-08) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +198/-1 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/M |
| Status | importado com ajustes |

## O que faz

Cacheia o resultado de parseRouterConfig por Ingress, com chave (namespace, nome, resourceVersion), faz deep copy na leitura e poda entradas não vistas no rebuild. Ingress sem mudança deixa de passar pelo decode por reflexão (paerser) a cada rebuild. O pprof do #13011 atribui ~19% do CPU de rebuild a esse ponto.

## Riscos e situação no upstream

Otimização de performance com estado no provider; o deep copy evita aliasing. Status 2-needs-review, sem review. Autor rebaseou para master a pedido do maintainer. Ganho mais relevante em clusters grandes.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/annotations.go`
- `pkg/provider/kubernetes/ingress/annotations_test.go`
- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13267/head:upstream-pr-13267`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- O cache por `namespace/name/resourceVersion`, podado a cada rebuild, é correto: o tamanho fica limitado ao número de Ingresses vivos. Só a goroutine de `Provide` acessa o cache, então não há corrida. As annotations só mudam junto com o `resourceVersion`.
- **Bug:** o `RouterIng.DeepCopy` escrito à mão não copiava o campo `RespondingTimeouts`, que existe no código atual mas não existia na base do PR. Os ponteiros `ReadTimeout`/`WriteTimeout` ficariam compartilhados entre o cache e a configuração emitida, e uma mutação posterior (por exemplo, no `routerTransform`) corromperia o cache entre rebuilds.
- Risco de manutenção: o `DeepCopy` é manual, então todo campo novo com ponteiro ou slice em `RouterIng` precisa ser incluído nele.

**Ajustes**

- `DeepCopy` passa a copiar `RespondingTimeouts`; o teste cobre isso.
- Conflitos resolvidos com a assinatura atual de `updateIngressStatus`; testes adaptados (`new(...)` e literais sem `ObjectMeta`, por causa do lint `embedlit`).

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress/` ok.
- `golangci-lint run ./pkg/provider/kubernetes/ingress/...`: 0 issues.
