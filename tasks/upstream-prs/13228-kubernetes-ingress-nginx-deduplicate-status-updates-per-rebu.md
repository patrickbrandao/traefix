# kubernetes ingress-nginx: deduplicate status updates per rebuild

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13228](https://github.com/traefik/traefik/pull/13228) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | MatteoManzoni |
| Aberto em | 2026-05-22 (última atualização: 2026-06-04) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +147/-16 em 3 arquivos |
| Labels | status/1-needs-design-review, size/M, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Deduplica as atualizações de status do provider ingress-nginx: indexa os Ingresses uma vez por rebuild e atualiza cada um no máximo uma vez. Também pula a fase de status quando publishService e publishStatusAddress não estão configurados. Não muda o resultado, só reduz chamadas à API em clusters grandes.

## Riscos e situação no upstream

Label needs-design-review com análise pedida pelo maintainer. Diff pequeno (+147/-16) em ingress-nginx/kubernetes.go e docs/content/migrate/v3.md.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/kubernetes.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `docs/content/migrate/v3.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13228/head:upstream-pr-13228`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
