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
| Status | adiado |

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

## Revisão (2026-11-16)

**Achados no PR original**

- O upstream está em `design-review` e o próprio autor diz que só completaria testes e documentação se a ideia fosse aceita. Não há testes nem documentação.
- O campo `RefreshSeconds` sai do `SwarmProvider` e vai para `Shared`. O nome e o JSON são preservados, então não quebra a configuração do Swarm, mas passa a existir no provider Docker com semântica diferente: só tem efeito quando `watch=true`. Com `watch=false`, que é justamente o caso em que um usuário esperaria polling, o provider continua lendo uma única vez e não atualiza mais.
- O refresh periódico roda dentro do mesmo loop de eventos, de forma síncrona. Um `ContainerList`/`inspect` lento atrasa o consumo de eventos. Não chega a ser um bug, mas falta definir o comportamento.
- Não há risco de segurança relevante: o padrão é 0 (desligado).

**Veredito:** adiado. A motivação (Podman sem eventos confiáveis) é legítima, mas falta definição de design no upstream e a semântica com `watch=false` está incompleta. A robustez do loop de eventos foi tratada separadamente no #13903 (importado); o #13578 (timeout do stream) também está adiado. Reavaliar quando o upstream decidir o design. Se for trazido, implementar o polling também para `watch=false` e documentar em `providers/docker.md`.

**Testes:** não aplicável (nada importado).
