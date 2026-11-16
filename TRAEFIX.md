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
  **GitHub Actions está desativado no repositório** durante o desenvolvimento inicial
  (reativar: `gh api -X PUT repos/patrickbrandao/traefix/actions/permissions -F enabled=true`).
  Requer os secrets `DOCKERHUB_USERNAME` e `DOCKERHUB_TOKEN` no repositório.
- Local: `scripts/traefix/docker-build.sh` (requer `docker login`).
- Build auto-contido em `Dockerfile.traefix` (compila WebUI + Go).

## Deploy rápido no servidor de testes

```bash
scripts/traefix/deploy-dev.sh            # build linux/amd64 + rsync + restart (~30s)
scripts/traefix/deploy-dev.sh rollback   # volta o binário anterior
scripts/traefix/deploy-dev.sh logs       # segue os logs
WEBUI=1 scripts/traefix/deploy-dev.sh    # recompila a WebUI também
```

O container do servidor monta o binário enviado em `/traefik` (bind mount); detalhes do
servidor em `.private/` (fora do git).

## Convenções

- Arquivos exclusivos do Traefix ficam separados dos do upstream (`Dockerfile.traefix*`,
  `scripts/traefix/`, `.github/workflows/traefix-*`, `tasks/`) para minimizar conflitos.
- `.private/` (ignorado pelo git) guarda dados sensíveis, como o servidor de testes.
