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
| `/api` | JSON do whoami com os campos novos `proto`, `prefix` e `time` (RFC 3339 UTC). |

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
2. **Sua conexão**: endereço e família (IPv4/IPv6), porta de origem, protocolo HTTP (`X-Traefix-Proto`),
   TLS (versão, cifra, grupo, SNI), latência (`rtt_us` do Transport ou `tcp_rtt_us` do Timing),
   saltos, sistema sugerido pelo TTL inicial, tipo de rede do navegador, ID da conexão.
3. **Seu navegador e dispositivo**: navegador, sistema, dispositivo/modelo, arquitetura, idiomas, fuso,
   tela, janela, tema, toque, núcleos, memória, GPU (WebGL), cookies. A latência é a do caminho
   (`rtt_min_us`, `tcp_rtt_us`…), não o RTT suavizado.
4. **Avaliação TCP/IP**: latência, MSS, escala de janela, SACK, timestamps, ECN, saltos, DF, opções IP,
   DSCP e reenvios, com `OK` / `Info` / `Atenção` / `Alerta` pelos critérios de
   [CRITERIA.md](CRITERIA.md). A maior parte é `Info`: descreve o sistema, não a qualidade.
5. **Sinais** (`OK` / `Atenção` / `Alerta`), coerência entre o que o navegador diz e o que a conexão
   mostra:
   - `navigator.webdriver` (automação);
   - versão do TLS; troca de chaves pós-quântica (`MLKEM`); sessão retomada; ECH aceito;
   - sistema do `User-Agent` contra `ttl_init` (64 = Linux/Android/macOS/iOS, 128 = Windows);
   - relay/proxy TCP: `rtt_gap_us` > 30 ms e > 50% de `tcp_rtt_us`, mais estrito que a regra da página
     *Fingerprint Signals* (15–20 ms e 30%, pensada para várias conexões agregadas);
   - Chromium sem Client Hints; GPC/DNT; cookies desativados; relógio com mais de 60 s de diferença
     (`Alerta` acima de 5 min).
6. **Impressões digitais**: JA4, JA3, JA4H, Akamai H2, JA4T, com explicação leiga (identificam o tipo
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
- Botões **Copiar JSON** e **Baixar JSON** (`{server, browser}`).

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
- Comparar o JA4 com uma base de navegadores conhecidos.
