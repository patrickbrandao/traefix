# Fix health check propagation from WRR children

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14261](https://github.com/traefik/traefik/pull/14261) |
| Tipo | Correção |
| Área | provider/kubernetes-crd (WRR, health check) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | rnikhori |
| Aberto em | 2026-11-10 (última atualização: 2026-11-10) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +160/-4 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/crd |
| Status | importado com ajustes |

## O que faz

Quando um IngressRoute tem vários services (ou TraefikService Weighted), o WRR pai nunca recebia o status dos filhos com healthCheck, então um backend DOWN continuava recebendo tráfego. O PR ativa HealthCheck no WRR gerado quando todos os filhos são Services Kubernetes com healthCheck.

## Riscos e situação no upstream

Base é v3.7, precisa de port para o master. Só altera o comportamento quando todos os filhos são Services Kubernetes com healthCheck, o que limita o escopo. Sem review de maintainer ainda (status/2-needs-review).

## Principais arquivos

- `pkg/provider/kubernetes/crd (geração do WRR pai)`
- `pkg/server/service/service.go (getWRRServiceHandler)`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14261/head:upstream-pr-14261`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:

- O bug é real (issue upstream #11622). O WRR pai, gerado pelo provider CRD (o implícito de uma rota com vários `services`, ou um `TraefikService` Weighted), nunca recebia `HealthCheck`. Por isso, os filhos `ExternalName` com `healthCheck` não se registravam como updaters do pai, e o tráfego continuava indo para um backend DOWN indefinidamente.
- A propagação só é ligada quando todo filho é um Kubernetes Service com `healthCheck`. É a única forma em que todos os filhos conseguem se registrar. Com um filho sem `healthCheck`, o build do serviço falharia com `cannot register ... as updater`. O fixture misto cobre esse caso.
- Mudança de comportamento: um WRR que cumpre essa condição passa a excluir filhos DOWN sem nenhuma mudança no YAML. É o comportamento esperado por quem configurou `healthCheck`, mas não estava documentado.
- Interação com o #13511: filhos com peso 0 continuam registrados como updaters. O WRR agora lida com isso sem loop infinito.
- Nada suspeito no diff.

Ajustes:

- Documentação: nota "Health Check Propagation" na seção Weighted Round Robin de `kubernetes/crd/http/traefikservice.md`, explicando quando a saúde dos filhos é levada em conta.
- Removido o comentário redundante ("o quê") no bloco que define `weighted.HealthCheck`. O comentário de "porquê" no início da função foi mantido.

Testes:

- `go test ./pkg/provider/kubernetes/crd/ ./pkg/server/service/`: ok.
- `golangci-lint run ./pkg/provider/kubernetes/crd/`: 0 issues.
- `go build ./...`: ok.
