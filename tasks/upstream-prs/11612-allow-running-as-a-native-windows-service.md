# Allow running as a native windows service

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11612](https://github.com/traefik/traefik/pull/11612) |
| Tipo | Recurso |
| Área | platform/windows |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | ahaenggli |
| Aberto em | 2025-03-15 (última atualização: 2026-03-01) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +462/-1 em 7 arquivos |
| Labels | kind/enhancement, status/2-needs-review, platform/windows, size/M |
| Status | pendente |

## O que faz

Faz o Traefik detectar quando roda como serviço nativo do Windows e só executar a lógica de serviço nesse caso, permitindo registrá-lo com sc.exe sem NSSM. Resolve a issue 10659.

## Riscos e situação no upstream

Código específico de Windows (7 arquivos, +462/-1); sem testes nem docs marcados. Label status/2-needs-review. Irrelevante para o Traefix rodando em container Linux.

## Principais arquivos

- `cmd/traefik/traefik.go`
- `pkg/winsvc/service.go`
- `pkg/winsvc/service_windows.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11612/head:upstream-pr-11612`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
