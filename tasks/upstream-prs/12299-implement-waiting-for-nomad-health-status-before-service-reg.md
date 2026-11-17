# Implement waiting for Nomad health status before service registration

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12299](https://github.com/traefik/traefik/pull/12299) |
| Tipo | Correção |
| Área | provider/nomad |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | kyounghoonJang |
| Aberto em | 2025-11-21 (última atualização: 2026-08-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +306/-29 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/provider/nomad |
| Status | importado com ajustes |

## O que faz

O provider Nomad registra serviços como saudáveis imediatamente, e o Traefik pode rotear para instâncias ainda não prontas. O PR passa a esperar o status de saúde do Nomad antes do registro. Resolve a issue 12295.

## Riscos e situação no upstream

Muda comportamento de registro do provider Nomad (6 arquivos, +306/-29). Review CHANGES_REQUESTED, saiu do milestone v3.7 por falta de tempo; usuários cobram andamento. Tem testes e fixtures.

## Principais arquivos

- `pkg/provider/nomad/nomad.go`
- `pkg/provider/nomad/config.go`
- `pkg/provider/nomad/nomad_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12299/head:upstream-pr-12299`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- O review do maintainer (CHANGES_REQUESTED) pedia opção para desligar o filtro e o fim do N+1; o autor resolveu as duas coisas (`disableHealthCheck` e uma única listagem de alocações por refresh).
- **Bug grave no modo `watch`:** o provider só assina o tópico `Service` do event stream. Como uma mudança de saúde da alocação não gera evento de serviço, um serviço registrado antes de ficar saudável nunca entraria na configuração, o que causaria indisponibilidade.
- **Bug com tasks de lifecycle:** qualquer task fora de `running` marcava a alocação inteira como não saudável. Tasks `prestart`/`poststart` que terminaram com sucesso ficam `dead` para sempre, então jobs com tasks de init seriam removidos do roteamento.
- `DeploymentStatus.Healthy == nil`, que é justamente o estado "ainda não saudável" durante um deployment, era tratado como saudável. Com isso o PR não resolvia o problema da issue (rotear para alocações ainda em health check).
- A listagem trazia todas as alocações do namespace, inclusive as terminais, a cada refresh.
- Os testes do PR passavam por *fail-open*: o mock respondia em `/v1/allocation/`, mas o cliente chama `/v1/allocations`. A listagem falhava, e o código marcava tudo como saudável.
- Logs `Warn` por serviço e por refresh, ruidosos em operação normal.

**Ajustes**

- Com o filtro ativo, `watch` passa a assinar também o tópico `Allocation`.
- Alocações listadas com o filtro `ClientStatus == "running"`. Uma alocação que não aparece na lista é tratada como não rodando.
- Saúde: se a alocação faz parte de um deployment, só entra quando o Nomad a marca como `Healthy=true`. Tasks `dead` sem `Failed` (hooks concluídos) são ignoradas; qualquer outro estado diferente de `running` filtra o serviço.
- Se a listagem falhar (ACL sem `read-job`, Nomad antigo), o filtro é pulado com um warning (*fail-open*, como no PR).
- Logs por serviço rebaixados para `Debug`. A função ficou pura (`isServiceHealthy`) e testável.
- Fixture `allocations.json` (lista) no lugar de `alloc_healthy.json`. Testes reescritos em tabela, mais um teste que verifica o filtro enviado ao Nomad e o efeito de `disableHealthCheck`.
- Documentação da opção em `providers/hashicorp/nomad.md`. `configuration-options.md` regenerado com `go run ./internal/`.

**Comportamento padrão:** o filtro fica ligado por padrão, como no upstream. Quem precisar do comportamento antigo usa `providers.nomad.disableHealthCheck=true`.

**Testes:** `go build ./...` ok; `go test ./pkg/provider/nomad/` ok; `golangci-lint run ./pkg/provider/nomad/...` com 0 issues.
