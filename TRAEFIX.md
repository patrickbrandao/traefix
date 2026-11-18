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

## Recursos de segurança

Especificação: `tmp/report-waf-features.md` (fases 0 a 4). Implementado até agora:

- **Fingerprint por entrypoint** (`entryPoints.<nome>.fingerprint`, fase 1): ID de conexão (ULID),
  JA3, JA4, Akamai h2 e JA4H calculados com a ordem real dos cabeçalhos em h1, h2 e h2c;
  cabeçalhos `X-Traefix-*` (prefixo em `headerPrefix`) nas requisições, com remoção dos
  enviados pelo cliente; exportação assíncrona em lote para uma URL no esquema
  `traefix.fingerprint/v1`; métricas `traefik_fingerprint_*` em todos os backends. h3 recebe
  ID, cabeçalhos e registros, sem JA4/JA4H por enquanto. Documentação:
  `docs/content/reference/install-configuration/entrypoints-fingerprint.md`.
- **forwardAuth com fail-open** (fase 2): `requestHeaders` (cabeçalhos fixos para o servidor de
  auth, tratados como segredo e recusados em `http://` público) e `failOpen` (erro de rede,
  timeout, 5xx e 4xx sem `denyRequiresResponseHeader` seguem ao backend com o cabeçalho
  marcador); métrica `traefik_forwardauth_failopen_total`. Código em
  `pkg/middlewares/auth/forward_failopen.go`; documentação em
  `docs/content/reference/routing-configuration/http/middlewares/forwardauth-failopen.md`.
- **Uso local dos fingerprints** (fase 3): campos `Fingerprint*` no access log e atributos de
  trace no span do router; matchers `JA3`, `JA4` e `JA4H` nas regras (router com erro se o
  entrypoint não tem `fingerprint`); middleware `fingerprintList` (modos `allow` e `deny`).
- **WAF Coraza nativo e QUIC** (fase 4): middleware `coraza` (`directives`, `crsEnabled`,
  `failOpen`), também lido como `plugin.coraza` quando nenhum plugin com esse nome está
  declarado; Coraza 3.8.1 e CRS 4.25.0 compilados no binário; métricas `traefik_waf_*`.
  QUIC: JA3/JA4 `q…` pelo ClientHello do handshake, ClientHello cru e transport parameters
  decifrando o Initial (com `rawBytes`), ordem dos cabeçalhos h3 e SETTINGS pelo qlog
  (`http3HeaderOrder`). Código em `pkg/middlewares/coraza/` e `pkg/fingerprint/quic*.go`.
- Código em `pkg/fingerprint/`; ganchos no upstream em `pkg/server/server_entrypoint_tcp*.go`,
  `pkg/server/router/router.go`, `pkg/config/static/entrypoints.go` e uma linha por backend em
  `pkg/observability/metrics/` (métricas em `traefix.go`).

## Convenções

- Arquivos exclusivos do Traefix ficam separados dos do upstream (`Dockerfile.traefix*`,
  `scripts/traefix/`, `.github/workflows/traefix-*`, `tasks/`) para minimizar conflitos.
- `.private/` (ignorado pelo git) guarda dados sensíveis, como o servidor de testes.
