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

## Forks de bibliotecas

As bibliotecas que o fingerprint usa são forks com o prefixo `traefix-`, ligados por `replace` no fim do
`go.mod`, com um agente responsável por cada um:

| Módulo | Fork | Agente | Mudança local |
|---|---|---|---|
| `github.com/quic-go/quic-go` | `patrickbrandao/traefix-quic-go` | `.claude/agents/traefix-quic-go.md` | Ordem dos SETTINGS do HTTP/3 no qlog; TTL e TOS do primeiro pacote (`ConnectionState.FirstPacket`) |
| `github.com/quic-go/qpack` | `patrickbrandao/traefix-qpack` | `.claude/agents/traefix-qpack.md` | Nenhuma |
| `golang.org/x/net` | `patrickbrandao/traefix-net` | `.claude/agents/traefix-net.md` | Nenhuma |

- Branch `traefix` = release oficial + commits "(Traefix)"; tags `<versão oficial>-traefix.<n>`.
- Clones locais em `~/Projects/traefix-forks/`, com o remote `upstream`.
- `scripts/traefix/check-forks.sh` exige que o `require` e a base do `replace` sejam a mesma versão; o
  `docker-build.sh` e o `deploy-dev.sh` rodam a verificação. Um bump vindo do Traefik oficial passa pelo
  agente do fork.
- Registros em `tasks/forks/<fork>/` (`INDEX.md`, `.last-sync`, propostas ao oficial).
- No Claude Code: _"use o agente traefix-quic-go para verificar o upstream"_.

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
- **Pedidos do finger-sniffer** (fase 5): `sender` e `traefik version` com `fork-point-<n>-g<sha>`,
  rótulos OCI e `NOTICE`; cabeçalho `<prefixo>Alpn`; `tls.ja3n`, `tls.ja4r`, `tls.ja4o` e
  `http2.headersPriority`; JSON Schema do lote
  (`docs/content/reference/install-configuration/traefix-fingerprint-v1.schema.json`, `v2` só com
  mudança incompatível) validado nos testes; goldens do finger-sniffer em
  `pkg/fingerprint/testdata/golden/`; opção `fingerprint.tcp` (Linux, `TCP_SAVE_SYN`): JA4T, assinatura
  p0f, TTL e RTT do handshake no registro, cabeçalho `<prefixo>Ja4t`, matcher `JA4T` e `ja4t` no
  `fingerprintList`; forwardAuth recusa valor vazio em `requestHeaders`.
- Código em `pkg/fingerprint/`; ganchos no upstream em `pkg/server/server_entrypoint_tcp*.go`,
  `pkg/server/router/router.go`, `pkg/config/static/entrypoints.go` e uma linha por backend em
  `pkg/observability/metrics/` (métricas em `traefix.go`).

## Cluster e certificados

Especificação: `tmp/report-cluster-certs.md` (três entregas: certificados em cluster, malha entre
servidores, satélite atrás de NAT). Implementado até agora:

- **Armazenamento dos certificados ACME** (`certificatesResolvers.<nome>.acme.certStore`, etapa 1 da
  primeira entrega): fonte da verdade em `file` (o `acme.json` de `storage`) ou `dir` (um diretório por
  domínio com `fullchain.pem`, `cert.pem`, `chain.pem`, `privkey.pem`, `cert.md5` e `cert.json`, no
  layout do projeto `traefik-certs`), com `export` para o outro. Com `primary: dir`, a conta fica em
  `<dir>/_traefix/<resolver>/account.json` e um `acme.json` existente é importado uma vez. Código em
  `pkg/provider/acme/store_*.go`; ganchos no upstream em `pkg/provider/acme/provider.go` (campo
  `CertStore`), `pkg/provider/acme/local_store.go` (cópia do registro antes de gravar),
  `pkg/config/static/static_config.go` e `cmd/traefik/traefik.go`. Documentação:
  `docs/content/reference/install-configuration/tls/certificate-resolvers/acme-certstore.md`.

## Infopage

`infopage/` (fork do `traefik/whoami`, imagem `tmsoftbrasil/traefix-infopage`, em
https://infopage.traefix.tmsoft.com.br) mostra ao visitante as informações do navegador e os cabeçalhos
do Traefix, nas abas "Resumo" e "Técnico", com uma pontuação de automação explicável (`infopage/SCORE.md`:
coerência UA × TLS × HTTP/2 × TCP, clientes medidos, listas de robôs, nuvens e iCloud Private Relay).
Especificação: `infopage/SPEC.md`.

**Todo recurso novo do Traefix transmitido em cabeçalhos deve ser propagado para o infopage** (regra de
propagação em `infopage/SPEC.md`).

## Convenções

- Arquivos exclusivos do Traefix ficam separados dos do upstream (`Dockerfile.traefix*`,
  `scripts/traefix/`, `.github/workflows/traefix-*`, `tasks/`, `infopage/`) para minimizar conflitos.
- `.private/` (ignorado pelo git) guarda dados sensíveis, como o servidor de testes.
