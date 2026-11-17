# Add docker provider polling

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12706](https://github.com/traefik/traefik/pull/12706) |
| Tipo | Recurso |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | karaolidis |
| Aberto em | 2026-02-19 (última atualização: 2026-06-02) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +18/-7 em 3 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, area/provider/docker, size/S |
| Status | pendente |

## O que faz

Adiciona polling opcional ao provider Docker, para ambientes como Podman em que os eventos de container não chegam ao Traefik. Resolve o caso de containers reiniciados que exigem reiniciar o Traefik.

## Riscos e situação no upstream

Pequeno (3 arquivos, +18/-7), mas sem testes nem docs (o autor marcou como pendentes). Status design-review. Polling aumenta carga na API do runtime.

## Principais arquivos

- `pkg/provider/docker/pdocker.go`
- `pkg/provider/docker/pswarm.go`
- `pkg/provider/docker/shared.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12706/head:upstream-pr-12706`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
