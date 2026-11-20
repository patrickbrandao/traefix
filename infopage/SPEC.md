# Traefix Infopage — especificação

Página que mostra a quem acessa tudo o que o navegador e a conexão revelam a um site: o que o próprio
navegador informa (APIs JavaScript) e o que o Traefix calcula da conexão e envia ao backend nos
cabeçalhos `X-Traefix-*`. Serve de vitrine e de ferramenta de teste para o fingerprint do Traefix.

- Código: esta pasta (`infopage/`), módulo Go próprio `github.com/patrickbrandao/traefix/infopage`,
  fora do módulo do Traefix (não entra no `go test ./...` da raiz).
- Origem: fork de [traefik/whoami](https://github.com/traefik/whoami) @ `1ce75d0`, renomeado de
  `whoami` para `infopage` em tudo (binário, módulo, serviço gRPC `infopage.Infopage`, variáveis
  `INFOPAGE_*`). Licença Apache 2.0 mantida; origem e alterações em `NOTICE`.
- Imagem: `docker.io/tmsoftbrasil/traefix-infopage` (pública), `linux/amd64` + `linux/arm64`.
- Produção de teste: https://infopage.traefix.tmsoft.com.br, container `traefix-infopage` no servidor
  de testes (script `setup-917-traefix-infopage.sh` em `tmsoft-servers/tmsoft-traefix`).

## Comportamento do servidor

Mantém todas as rotas do whoami (`/`, `/api`, `/bench`, `/data`, `/echo`, `/health`, `/openapi.yaml`,
gRPC). Acrescenta:

| Rota | Comportamento |
|---|---|
| `/` | `GET` com `Accept` contendo `text/html` (navegador) recebe a página; os demais (curl, scripts) recebem o texto do whoami. `?format=html` ou `?format=text` força um dos dois. |
| `/assets/*` | `app.js` e `style.css` embutidos no binário (`static/`, `go:embed`). Sem listagem de diretório. |
| `/api` | JSON do whoami com os campos novos `proto`, `prefix`, `time` (RFC 3339 UTC) e `score` (pontuação passiva). |
| texto (curl) | Acrescenta `Automation-Score`, `Automation-Known-TLS` e uma linha `Automation-Rule` por regra com pontos. |

- A página recebe os dados da **própria requisição de navegação**, embutidos no HTML como JSON
  (`<script id="infopage-data" type="application/json">`), e não de um `fetch` posterior: o `fetch` tem
  outros `Accept`, `Sec-Fetch-*` e outro JA4H.
- Opção `-prefix` / `INFOPAGE_HEADER_PREFIX` (padrão `X-Traefix-`): o mesmo valor de
  `fingerprint.headerPrefix` do entrypoint.
- Segurança: CSP `default-src 'none'` com `script-src`/`style-src 'self'`, `no-store`, `nosniff`,
  `Referrer-Policy: no-referrer`. O JSON escapa `<`, `>` e `&`; o JavaScript escreve tudo com
  `textContent`, nunca `innerHTML`.

## Página

HTML, CSS e JavaScript simples e estáticos, sem framework nem build, sem recursos externos. Largura
fixa de 880 px centralizada (fluida abaixo disso), tema claro/escuro por `prefers-color-scheme`. Textos
em português do Brasil. A aba ativa fica no fragmento (`#tecnico`).

### Aba "Resumo" (para pessoas)

1. **Destaque**: IP, navegador (nome e versão principal) e sistema operacional.
2. **Pontuação de automação** ([SCORE.md](SCORE.md)): índice 0–100 não calibrado, faixa, papel (navegador,
   navegador por proxy ou SOCKS, automação declarada, robô verificado ou não), cobertura, TLS reconhecido, relay/datacenter/robô pelo
   endereço; tabela com **todos** os itens analisados, agrupados por família (valor observado, esperado,
   resultado e pontos), a soma bruta e a soma com teto de cada família, o total `S`, o índice `R` com a fórmula
   e a cobertura; no fim, a **avaliação completa** (resultado, papel, itens com e sem dado, o que pesou para cada
   lado, tetos aplicados).
3. **Sua conexão**: endereço e família (IPv4/IPv6), porta de origem, protocolo HTTP (`X-Traefix-Proto`),
   TLS (versão, cifra, grupo, SNI), latência (`rtt_us` do Transport ou `tcp_rtt_us` do Timing),
   saltos, sistema sugerido pelo TTL inicial, tipo de rede do navegador, ID da conexão.
4. **Linha do tempo da conexão** (TCP): SYN (estimado: aceite menos a volta TCP), ACK, primeiro byte do
   ClientHello, ClientHello completo, resposta do servidor, Finished do cliente, fim do handshake e a requisição,
   cada um com o horário UTC em microssegundos (`accepted` do `X-Traefix-Transport` mais os tempos do
   `X-Traefix-Timing`), a distância desde o SYN e a diferença para o marco anterior; barra proporcional; volta TCP,
   volta TLS, diferença e atraso do primeiro byte com os limiares; tempos do navegador (Navigation Timing) só para
   comparação; diagnóstico "Conexão direta", "Indício fraco", "Proxy ou SOCKS no caminho" ou "Relay de
   privacidade" pela regra TIME-RELAY ([SCORE.md](SCORE.md) §4.5). Em HTTP/3, o Initial, o fim do handshake QUIC e
   a requisição.
5. **MTU estimado** ([CRITERIA.md](CRITERIA.md), "MTU estimado"): base (MSS do SYN ou `max_udp_payload` do
   QUIC), + TCP 20 ou UDP 8, + IPv4 20 ou IPv6 40, nota de 0 a 100 pela régua (1500 = 100, PPPoE 1480–1492 =
   90–95, 1340 = 0) e a régua desenhada. Não entra na pontuação de automação.
6. **Seu navegador e dispositivo**: navegador, sistema, dispositivo/modelo, arquitetura, idiomas, fuso,
   tela, janela, tema, toque, núcleos, memória, GPU (WebGL), cookies. A latência é a do caminho
   (`rtt_min_us`, `tcp_rtt_us`…), não o RTT suavizado.
7. **Avaliação TCP/IP**: latência, MTU estimado, escala de janela, SACK, timestamps, ECN, saltos, DF, opções IP,
   DSCP e reenvios, com `OK` / `Info` / `Atenção` / `Alerta` pelos critérios de
   [CRITERIA.md](CRITERIA.md). A maior parte é `Info`: descreve o sistema, não a qualidade.
8. **Sinais** (`OK` / `Atenção` / `Alerta`), coerência entre o que o navegador diz e o que a conexão
   mostra:
   - `navigator.webdriver` (automação);
   - versão do TLS; troca de chaves pós-quântica (`MLKEM`); sessão retomada; ECH aceito;
   - sistema do `User-Agent` contra `ttl_init` (64 = Linux/Android/macOS/iOS, 128 = Windows);
   - relay/proxy TCP: `rtt_gap_us` > 30 ms e > 50% de `tcp_rtt_us`, mais estrito que a regra da página
     *Fingerprint Signals* (15–20 ms e 30%, pensada para várias conexões agregadas);
   - Chromium sem Client Hints; GPC/DNT; cookies desativados; relógio com mais de 60 s de diferença
     (`Alerta` acima de 5 min).
9. **Impressões digitais**: JA4, JA3, JA4H, Akamai H2, JA4T, com explicação leiga (identificam o tipo
   de cliente, não a pessoa). Sem cabeçalhos do Traefix, explica que o fingerprint não está ativo.

### Aba "Técnico" (todos os detalhes)

- **Requisição**: método, URL, host, protocolo no backend, no proxy e no navegador (`nextHopProtocol`),
  RemoteAddr.
- **Cabeçalhos do Traefix** (`<prefixo>*`, ordem alfabética): todo cabeçalho com o prefixo aparece,
  mesmo os que a página ainda não conhece. Valores no formato `chave=valor,…` (Tls, Transport, Timing,
  H2-Detail, H3, Tls-Hello) viram sub-tabela automaticamente.
- **Demais cabeçalhos**: na ordem enviada pelo cliente quando há `<prefixo>Header-Order`.
- **Navegador**: `navigator`, Client Hints de alta entropia, `Intl`, tela, media queries, Network
  Information, Navigation Timing, WebGL, recursos, armazenamento, relógio (diferença para o servidor).
- **Servidor infopage**: nome, hostname, IPs.
- **Regras da pontuação**: todas as regras avaliadas, com valor, esperado, estado e pontos.
- Botões **Copiar JSON** e **Baixar JSON** (`{server, browser, score}`).

## Pontuação de automação

Especificação e pesos em [SCORE.md](SCORE.md). O servidor calcula as regras passivas (`score.go`) sobre os
cabeçalhos da requisição e as embute em `infopage-data`; o `app.js` acrescenta a família `js` e agrega com os
mesmos tetos e a mesma calibração.

Listas de endereços (`lists.go`), baixadas no início e a cada 24 h (1 h após falha), mantendo a cópia anterior
quando o download falha ou encolhe mais da metade; desligadas com `INFOPAGE_LISTS=0`:

- robôs verificáveis: Google (3 listas), Bing, Apple, OpenAI (GPTBot, OAI-SearchBot, ChatGPT-User),
  DuckDuckGo, Perplexity (2), Common Crawl; Google, Bing e Apple também por DNS reverso e direto (cache 24 h);
- nuvens: AWS, Google Cloud, Oracle, DigitalOcean, Cloudflare;
- iCloud Private Relay (exceção: TCP, TTL e tempo do relay não contam).

Memória: as faixas ficam em 16 bytes por endereço (285 mil só no Private Relay), os downloads são lidos em
streaming e o processo usa `GOMEMLIMIT` de 40 MiB por padrão; no container de 64 MB, pico de 46 MB na
atualização e 37 MB estável (medido com `docker run --memory=64m`).

Clientes medidos (JA4) vêm da base própria do finger-sniffer (`known_clients.json`) e dos goldens.

## Regra de propagação

**Todo recurso novo do Traefix transmitido em cabeçalhos deve ser propagado para o infopage** no mesmo
trabalho que o cria:

1. Aba "Técnico": já aparece sozinho (cabeçalhos com o prefixo). Conferir se o formato é legível; um
   formato novo de valor precisa de parser em `app.js`.
2. Aba "Resumo": quando o valor interessa a uma pessoa, acrescentar a linha amigável (cartão "Sua
   conexão" ou "Impressões digitais"); se ele tem avaliação, a regra em "Avaliação TCP/IP" com o critério
   e a fonte registrados em [CRITERIA.md](CRITERIA.md); se for um sinal de coerência, a regra em
   "Sinais", com o critério da documentação `entrypoints-fingerprint-signals.md`.
3. Atualizar esta especificação, publicar a imagem (`scripts/traefix/infopage-build.sh`) e recriar o
   container no servidor de testes.

## Build e publicação

```bash
cd infopage && make test check               # testes e golangci-lint
make run                                     # local em http://localhost:8080
scripts/traefix/infopage-build.sh            # multi-arch, push :<versão> e :latest
```

No servidor: `bash /root/setup-917-traefix-infopage.sh` (puxa `:latest` e recria o container com as
labels do Traefik).

## Próximos passos

- Sinais que dependem de várias conexões (TCP contra HTTP/3 do mesmo cliente).
- Botão para repetir a medição em HTTP/3 (o navegador só usa h3 após o `Alt-Svc`).
- Medir Windows, Linux, Android, iOS, Safari e Edge para a base de clientes conhecidos.
- ASN, Tor e reputação (iptoasn, lista de saídas Tor, Spamhaus DROP) na família `net`.
