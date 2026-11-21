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

- **Base do cluster** (`cluster`, etapa 2): cada instância se anuncia num Valkey ou Redis (único,
  Sentinel ou Cluster) com `nodeID` e `priority`; o pacote `pkg/cluster/` oferece a lista de nós vivos,
  travas com validade (`TryLock`, `Keep`, `Release`) e um canal de eventos com reassinatura. Ganchos no upstream em `pkg/config/static/static_config.go` (campo
  `Cluster`) e `cmd/traefik/traefik.go`. Documentação:
  `docs/content/reference/install-configuration/cluster.md`.

- **Certificados no Valkey** (`certStore.primary: redis`, etapa 3): conta e certificados do resolver ficam
  no servidor do `cluster`, em `traefix:acme:{<resolver>}:*`; cada instância mantém uma cópia em memória,
  recebe os certificados das outras por evento e por comparação a cada 15 s, grava as suas exportações e
  parte delas quando o servidor está fora. A cópia `tcerts:cert:<domínio>` atende o `cert-get` e o
  `cert-watcher` do `traefik-certs`. Código em `pkg/provider/acme/store_redis.go` e `provider_store.go`; ganchos no upstream em
  `Provide` e `getClient` de `pkg/provider/acme/provider.go`.

- **Eleição dos pedidos à CA** (`certStore.election`, etapa 4): para cada certificado, uma instância pede
  à CA, a de maior `cluster.priority` entre as vivas com o resolver; empate decide na corrida pela trava,
  e `priority: 0` nunca pede. Qualquer instância anuncia o que os seus routers precisam
  (`traefix:acme:{<resolver>}:want:<id>`); as falhas são compartilhadas, com espera crescente (2 min a
  6 h) e desistência após 14 dias; as renovações e o registro da conta seguem a mesma eleição. Código em
  `pkg/provider/acme/coordinator.go` e `provider_coordinator.go`; no upstream, `resolveCertificate` e `resolveDefaultCertificate` passaram a se
  chamar `obtainCertificate` e `obtainDefaultCertificate`, e `renewCertificates` chama `electRenewals`.

- **Desafio HTTP-01 compartilhado** (etapa 5): com `cluster`, a instância que pede o certificado grava a
  resposta do desafio em `traefix:acme:http:<token>:<domínio>` (10 min) e qualquer instância atrás do nome
  responde à validação da CA, consultando o Valkey só para tokens com formato de ACME (500 ms de limite,
  5 s de memória para o que não existe, 32 consultas simultâneas). Vale para todos os resolvers da
  instância, e toda réplica que recebe o tráfego precisa do `httpChallenge`, inclusive as de
  `priority: 0`. Código em `pkg/provider/acme/challenge_http_cluster.go`; gancho no upstream em
  `cmd/traefik/traefik.go` (`acme.NewHTTPChallengeSolver`).

- **Renovação e Valkey fora do ar** (etapa 6): as renovações de uma verificação passam pela eleição em
  lote (uma espera só), e a instância que acha certificados a renovar anuncia a lista no evento `renew`;
  as de prioridade maior reservam os que têm na mesma versão antes da sua própria verificação. Com o
  Valkey fora, nenhuma chamada espera por ele: os pedidos ficam pendentes e voltam quando ele volta. Um
  certificado obtido por outra conta ACME (importado do `acme.json` de outra instância) é renovado sem o
  vínculo ARI quando a CA recusa a substituição (`pkg/provider/acme/provider_renew.go`; gancho de uma
  linha em `renewCertificates`).

- **Métricas e teste de ponta a ponta** (etapa 7): `traefik_cluster_redis_up`, `traefik_cluster_nodes`,
  `traefik_acme_cluster_elections_total`, `traefik_acme_cluster_issuances_total`,
  `traefik_acme_cluster_challenge_lookups_total`, `traefik_acme_cluster_config_mismatch` e
  `traefik_acme_store_writes_total`, em `pkg/observability/metrics/traefix_cluster.go`.
  `scripts/traefix/e2e-cluster.sh` confere os cenários de aceite com o binário real, um `redis-server` e
  o Pebble, sem Docker (`go install github.com/letsencrypt/pebble/v2/cmd/pebble@latest`).

## Malha entre servidores

Especificação: `tmp/report-cluster-mesh.md` (segunda entrega, nove etapas, todas feitas). Cada Traefix anuncia aos outros
os routers que descobriu, e uma requisição que chega a um servidor sem o container é encaminhada ao
servidor que o tem, sem rede overlay. Implementado até agora:

- **Configuração e identidade** (`cluster.mesh`, etapa 1): `cluster.token` (ou `tokenFile`, 32 caracteres
  ou mais) e `additionalTokens` para a troca de token; `cluster.redis` passou a ser opcional (um cluster
  precisa de `redis`, de `mesh` ou dos dois). Cada instância deriva do token uma autoridade certificadora
  (PBKDF2-HMAC-SHA256, 600 mil iterações, com o `prefix` no sal) e assina o seu certificado; as instâncias
  com o mesmo token chegam à mesma autoridade e se verificam por TLS 1.3 mútuo, sem o token trafegar. Os
  certificados não vencem e são conferidos numa data fixa, então os relógios não precisam concordar. A
  impressão digital da autoridade sai no log e no anúncio da instância no Valkey (`mesh.authority`, com
  `mesh.addr`). Código em `pkg/mesh/` e `pkg/cluster/mesh.go`; gancho no upstream em
  `cmd/traefik/traefik.go`. Documentação:
  `docs/content/reference/install-configuration/cluster-mesh.md`.

- **Ligação** (etapa 2): cada instância escuta em `cluster.mesh.address` (UDP e TCP) e disca para os
  endereços de `cluster.mesh.peers`. A ligação é HTTP/3 sobre QUIC, com reserva em HTTP/2 sobre TLS/TCP,
  sob o ALPN próprio `traefix-mesh/1`; quem abre a ligação envia as requisições, então um par tem duas.
  Apresentação (`/v1/hello`), ping três vezes por `linkTimeout`, rediscagem de 1 s a 30 s, volta ao QUIC
  a cada 5 min. Os túneis que as próximas etapas usam já existem e estão testados nos dois transportes:
  stream por `CONNECT` (`session.OpenStream`) e fluxo de pacotes por datagramas QUIC com sobra pelo
  stream (`session.OpenFlow`). Métricas `traefik_mesh_peers`, `traefik_mesh_link_up` e
  `traefik_mesh_link_rtt_seconds`.

- **Descoberta** (etapa 3): um registro de pares (`pkg/mesh/peers.go`) com cinco fontes: `mesh.peers`,
  o anúncio das instâncias no Valkey (`mesh.addr`, recusando quem tem outro `mesh.authority`), DNS
  (`mesh.dns`: SRV e TXT `traefix-mesh=`, a cada minuto), a troca entre pares (o `hello` leva os
  endereços de quem disca e os pares com que ele está em contato; `/v1/peers` a cada 30 s) e o
  `mesh.peersFile`. Quem recebe uma ligação liga de volta, então uma instância nova que conhece uma só
  acaba ligada a todas. Um par tem uma ligação só, com vários endereços tentados em ordem. Pares
  aprendidos saem após `peerRetention` sem contato; os de `mesh.peers` nunca.

- **Rotas** (etapa 4): campos `mesh.priority` e `mesh.export` nos routers HTTP, TCP e UDP
  (`pkg/config/dynamic/mesh_traefix.go`; campo `Mesh` no fim de `Router`, `TCPRouter` e `UDPRouter`, e o
  deepcopy regenerado com `deepcopy-gen`). A malha lê os routers dos providers locais por um
  transformador do `ConfigurationWatcher`, antes de os middlewares do entrypoint entrarem, e anuncia
  regra, prioridade, entrypoints e TLS. Cada ligação acompanha a tabela do outro lado por consulta longa
  (`/v1/routes`, com geração). O provider `mesh` cria um router `<router>-<provider>@mesh` para cada
  router HTTP anunciado que a instância não tem, nos entrypoints de mesmo nome. `GET /api/mesh` mostra
  ligações, rotas exportadas e o que foi feito das importadas. Ganchos no upstream:
  `cmd/traefik/traefik.go` (duas chamadas; o resto em `traefix_mesh.go`), `pkg/api/handler.go`
  (`appendTraefixRoutes`) e `pkg/server/service/managerfactory.go` (`traefixBuilders`).

- **Encaminhamento HTTP** (etapa 5): o serviço de um router `@mesh` envia a requisição pela ligação a
  uma instância dona do router (maior `mesh.priority`; no empate, `balance`). A requisição leva o
  cabeçalho `Traefix-Mesh` (base64 de JSON: router, endereço do visitante, TLS e fingerprint). O dono a
  entrega ao router pelo nome, com os middlewares próprios do router e o serviço, sem casar a regra nem
  rodar os middlewares dos seus entrypoints (`Manager.BuildNamedHandlers`, em
  `pkg/server/router/router_traefix_mesh.go`; entrypoint `traefix-mesh` no access log). O dono enxerga o
  endereço do visitante, um estado TLS sintético e o fingerprint medido na entrada
  (`fingerprint.ContextWithForwarded`). `Upgrade` (WebSocket) vai por um stream `CONNECT` servido como
  conexão HTTP/1.1; gRPC e fluxos nos dois sentidos passam pela ligação. Respostas da malha levam
  `Proxy-Status`. Métricas `traefik_mesh_requests_total` e `traefik_mesh_received_total`. Código em
  `pkg/mesh/forward.go`; gancho no upstream em `pkg/server/routerfactory.go` (`buildMeshHandlers`).

- **Prioridade** (etapa 6): quando a instância também tem o router, um ponto de decisão entra entre os
  middlewares do entrypoint e os do router (`Manager.buildMiddlewaresAndService`, gancho de uma linha em
  `buildHTTPHandler` de `pkg/server/router/router.go`; `RouterFactory.setMeshDecider`). A escolha
  (`Mesh.plan`, em `pkg/mesh/forward.go`): maior `mesh.priority` entre os candidatos disponíveis; na
  prioridade padrão a instância prefere a si mesma; no empate vale o `balance`. O dono anuncia `down`
  quando o serviço do router não tem servidor de pé (`pkg/mesh/health.go`, conferido a cada segundo no
  estado dos servidores do `runtime.Configuration`). Três falhas seguidas põem um par em quarentena para a
  rota (5 s, dobrando até 60 s). Uma requisição que não saiu da instância (sinal `WroteHeaders` do
  `httptrace`), ou que o dono disse não ter atendido e não tem corpo, vai ao candidato seguinte.

- **TCP** (etapa 7): os routers TCP anunciados viram routers `<router>-<provider>@mesh` com a regra, a
  prioridade e o TLS (`passthrough` ou TLS terminado na entrada); cada conexão vai num stream `CONNECT` da
  ligação, com meio-fechamento nos dois sentidos, e o dono a entrega ao router pelo nome, com os seus
  middlewares TCP e o serviço, vendo o endereço do visitante e o de destino (PROXY protocol ao container com o
  IP certo). Prioridade, saúde (health check TCP), quarentena e nova tentativa seguem as regras do HTTP; uma
  conexão sobrevive às recargas da configuração. As opções TLS sem provider são anunciadas com o provider do
  router (`modern` vira `modern@file`). Código em `pkg/mesh/tcp.go`; ganchos no upstream em
  `pkg/server/router/tcp/manager.go` (`buildMiddlewaresAndService` e o nome do router no contexto, com o resto
  em `manager_traefix_mesh.go`), `pkg/server/service/tcp/service.go` (`buildTraefix`, em
  `service_traefix.go`) e `pkg/server/routerfactory.go` (`setMeshTCP`, `switchMeshHandlers`).

- **UDP** (etapa 8): os routers UDP anunciados viram routers `<router>-<provider>@mesh` só nos entrypoints UDP sem
  router próprio (um entrypoint tem um router UDP só; o primeiro pelo nome leva). Cada sessão UDP vai num fluxo da
  ligação (`session.OpenFlow`: datagramas QUIC, com sobra pelo stream para o que não cabe e tudo pelo stream na
  reserva em TCP). No dono, o fluxo vira uma sessão `udp.Conn` sem socket (`udp.NewFlowConn`, em
  `pkg/udp/conn_traefix.go`, com ganchos de três linhas em `Write` e `Close` de `pkg/udp/conn.go`) entregue ao
  serviço do router pelo nome. A entrada decide o fim da sessão (`udp.timeout` do entrypoint); o dono espera esse
  tempo mais o `linkTimeout`. Prioridade e quarentena como no HTTP; o router UDP fica fora quando o serviço não tem
  servidor. Código em `pkg/mesh/udp.go`; ganchos no upstream em `pkg/server/router/udp/router.go` (`meshHandler`,
  com o resto em `manager_traefix_mesh.go`), `pkg/server/service/udp/service.go` (`buildTraefix`, em
  `service_traefix.go`) e `pkg/server/routerfactory.go` (`setMeshUDP`).

- **Acabamento** (etapa 9): na apresentação, cada instância informa por entrypoint o protocolo, se tem
  `fingerprint` e a lista de middlewares; as de mesmo nome que diferem geram um aviso no log, o campo
  `configMismatch` de `/api/mesh` e a métrica `traefik_mesh_config_mismatch` (`pkg/mesh/config.go`; gancho em
  `cmd/traefik/traefix_mesh.go`). Métricas `traefik_mesh_routes`, `traefik_mesh_exported_routes` e
  `traefik_mesh_udp_packets_total`. `scripts/traefix/e2e-mesh.sh` confere com o binário real, sem Docker, os
  cenários `formation token http priority tcp udp fallback config` (55 verificações, cerca de 1 min;
  `TRANSPORT=tcp` para as ligações pela reserva), com os servidores de teste em `scripts/traefix/e2e-mesh/`.

## Satélite atrás de NAT

Especificação: `tmp/report-cluster-satellite.md` (terceira entrega, sete etapas). Um satélite é um Traefix sem conexão de
entrada (NAT doméstico, CGNAT): disca para os centrais e recebe o tráfego pelo que abriu. Usa o token do cluster, anuncia
os seus routers, não importa nenhum, não usa Valkey e não pede certificado à CA. Implementado até agora:

- **Configuração e papel** (etapa 1): `cluster.mesh.satellite` (exige `peers` ou `dns`, recusa `advertise` e
  `cluster.redis`); `role` (`member` ou `satellite`) na apresentação e em `/api/mesh`; os centrais não discam para um
  satélite nem o passam adiante, um endereço de satélite numa lista de pares é largado com aviso, e satélite não liga a
  satélite; o satélite não consulta as rotas dos outros; os resolvers ACME do satélite não iniciam, nem o router
  `acme-http@internal`, e os nomes deles continuam valendo nos routers. Código em `pkg/mesh/satellite.go`; gancho no
  upstream em `cmd/traefik/traefik.go` (`satelliteResolvers`, em `traefix_mesh.go`). Documentação:
  `docs/content/reference/install-configuration/cluster-satellite.md`.
- **Socket único** (etapa 2): toda instância escuta e disca por QUIC pelo mesmo `quic.Transport`
  (`pkg/mesh/listener.go`, `dialQUICFrom` em `transport.go`), então as ligações saem da porta da malha. É a condição
  para um central voltar ao satélite pelo mapeamento do NAT. **Os centrais ainda não alcançam o satélite de volta**
  (etapa 3).

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
