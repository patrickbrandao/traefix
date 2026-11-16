# Traefix

Fork privado do [Traefik](https://github.com/traefik/traefik), mantido em
`github.com/patrickbrandao/traefix`.

## Remotes

| Remote     | URL                                              |
|------------|--------------------------------------------------|
| `origin`   | https://github.com/patrickbrandao/traefix.git    |
| `upstream` | https://github.com/traefik/traefik.git           |

Ponto do fork: tag `fork-point` (upstream `master` @ `36b2bd1`, 2026-11-13, base `v3.8.0-rc.3`).

## Acompanhamento do upstream

O agente `upstream-tracker` (`.claude/agents/upstream-tracker.md`) busca novidades do oficial
desde `tasks/upstream/.last-sync` e gera tasks em `tasks/upstream/`. **Alterações locais do
Traefix têm prioridade** sobre mudanças do oficial.

No Claude Code: _"use o agente upstream-tracker para verificar o upstream"_.

## Imagens Docker

`docker.io/tmsoftbrasil/traefix` (pública), `linux/amd64` + `linux/arm64`.

- CI: `.github/workflows/traefix-docker.yaml` (push em `master`, tags `v*`, ou manual).
  Requer os secrets `DOCKERHUB_USERNAME` e `DOCKERHUB_TOKEN` no repositório.
- Local: `scripts/traefix/docker-build.sh` (requer `docker login`).
- Build auto-contido em `Dockerfile.traefix` (compila WebUI + Go).

## Convenções

- Arquivos exclusivos do Traefix ficam separados dos do upstream (`Dockerfile.traefix*`,
  `scripts/traefix/`, `.github/workflows/traefix-*`, `tasks/`) para minimizar conflitos.
- `.private/` (ignorado pelo git) guarda dados sensíveis, como o servidor de testes.
