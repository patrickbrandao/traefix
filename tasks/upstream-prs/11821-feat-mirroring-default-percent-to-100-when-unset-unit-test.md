# feat(mirroring): default percent to 100 when unset + unit test

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11821](https://github.com/traefik/traefik/pull/11821) |
| Tipo | Correção |
| Área | middleware/mirroring, provider/k8s/crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | Alexy-vda |
| Aberto em | 2025-06-05 (última atualização: 2025-06-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +67/-16 em 8 arquivos |
| Labels | status/2-needs-review, size/S, area/provider/k8s/crd, area/provider/k8s, breaking |
| Status | adiado |

## O que faz

Quando percent não é informado em mirroring, o padrão passa a ser 100, em vez de nenhum tráfego espelhado. Corrige o comportamento confuso descrito na issue 11818.

## Riscos e situação no upstream

Mudança breaking: campo Percent vira *int. Um maintainer comentou que faz sentido mas só será mergeado no próximo major (v4). Não aplicar antes disso sem decisão explícita.

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/server/service/service.go`
- `pkg/provider/kubernetes/crd/kubernetes_http.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11821/head:upstream-pr-11821`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.**

Achados:
- Mudança breaking: `MirrorService.Percent` vira `*int` e configurações existentes sem `percent` (hoje 0% espelhado) passariam a espelhar 100% do tráfego. Isso pode mandar requests (com corpos e credenciais) para um backend de espelho que o operador não esperava receber tráfego. O maintainer (nmengin) disse que só entra no próximo major.
- No provider CRD o PR usa `Percent: &mirror.Percent` sobre um `int` do CRD, então o default nunca é aplicado no Kubernetes (ponteiro para 0 sempre); o comportamento fica inconsistente entre providers.
- `zz_generated.deepcopy.go` não foi regenerado para o novo campo ponteiro.

Motivo: esperar a decisão do upstream para o v4; mudar o default de mirroring no fork sem isso tem risco de vazamento de tráfego para espelhos.

Testes: nenhum (não importado).
