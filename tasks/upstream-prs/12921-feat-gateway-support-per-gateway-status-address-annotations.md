# feat(gateway): support per-Gateway status address annotations

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12921](https://github.com/traefik/traefik/pull/12921) |
| Tipo | Recurso |
| Área | provider/kubernetes/gateway-api |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jscaltreto |
| Aberto em | 2026-04-01 (última atualização: 2026-05-20) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +402/-12 em 5 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/provider/k8s/gatewayapi |
| Status | importado com ajustes |

## O que faz

Permite sobrescrever o statusAddress estático por Gateway através de annotations (ip, hostname, service), espelhando as opções estáticas. Assim, Gateways atrás de Services diferentes (interno/público) passam a reportar endereços distintos em .status.addresses, o que o external-dns precisa.

## Riscos e situação no upstream

Label status/1-needs-design-review; maintainer abriu design review em abril e não houve revisão de código depois; autor vem cobrando. Toca annotations.go e kubernetes.go do provider gateway, sem conflito previsto com o fork.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/annotations.go`
- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `pkg/provider/kubernetes/gateway/annotations_test.go`
- `docs/content/reference/routing-configuration/kubernetes/gateway-api.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12921/head:upstream-pr-12921`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:
- Uma annotation inválida (IP vazio, Service inexistente, ReferenceGrant ausente) fazia `continue` no loop de Gateways: o Gateway inteiro sumia da configuração e o tráfego dele parava, só por causa de um problema de status. Um erro de digitação do dono do Gateway derrubaria o roteamento.
- O IP da annotation não era validado; como o valor vem do dono do Gateway (não do operador), um valor arbitrário iria para `.status.addresses` e, via external-dns, para o DNS.
- Referência cross-namespace a Service: o PR exige ReferenceGrant (`from: Gateway`, `to: Service`), o que é correto; o static config continua isento por ser do operador. Sem teste para esse caminho.
- Conflitos com o fork (ListenerSets, `gatewayWithListeners`, `new(...)` em vez de `ptr.To`).

Ajustes:
- Erro nas annotations agora só é logado e o Gateway usa o `statusAddress` da configuração de instalação; o roteamento não é afetado.
- Validação de IP com `net.ParseIP`.
- Testes novos: IP inválido e Service cross-namespace sem ReferenceGrant; testes adaptados ao estilo do fork.
- Docs mencionam o fallback em caso de annotation inválida.

Risco residual: o dono de um Gateway passa a controlar o endereço publicado no status do próprio Gateway (antes só o operador). Rotas de outros namespaces que se anexam a esse Gateway já confiam nele, então o impacto fica restrito.

Testes: `go test ./pkg/provider/kubernetes/gateway/` ok; `golangci-lint run ./pkg/provider/kubernetes/gateway/...` 0 issues.
