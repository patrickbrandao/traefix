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
| Status | pendente |

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
