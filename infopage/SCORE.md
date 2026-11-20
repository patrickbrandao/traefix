# Pontuação de automação do infopage

Especificação de 2026-11-19, implementada em `score.go` (regras passivas), `lists.go` (listas de endereços) e
`static/assets/app.js` (família `js` e agregação). Base: `tmp/antibot-report.md` (consolidação de nove pesquisas),
`tmp/bot-detect-methods.md` (bases do finger-sniffer) e o que o Traefix entrega nos cabeçalhos `X-Traefix-*`. Complementa [CRITERIA.md](CRITERIA.md): aquele
documento diz se um valor é **bom para o usuário**; este diz quanto um valor **contribui para tratar o
acesso como automação**. São dois eixos diferentes e o infopage deve mostrar os dois.

## 1. Princípios tirados do relatório

1. **Um sinal vale pelo que contradiz, não pelo que é.** Um JA4 desconhecido não é suspeito, é
   desconhecido. Um JA4 de OpenSSL com `User-Agent` de Chrome é suspeito porque contradiz a declaração.
   Quase todos os pontos nascem do **delta entre o que o cliente declara** (UA, Client Hints, navigator)
   e **o que as camadas de baixo mostram** (TCP, TLS, HTTP/2, ordem dos cabeçalhos).
2. **Fora do ideal ≠ bot.** MSS 1380, latência de 250 ms, relógio adiantado, TLS sem retomada: são
   qualidade de rede ou do dispositivo, valem `0` ponto. A coluna "ideal" e a coluna "pontos" são
   independentes.
3. **Ausência não pontua.** Valor que o Traefix não mediu (`tcp` desligado, JA4H em h3 sem
   `http3HeaderOrder`, JS bloqueado) vira `desconhecido` ou `não aplicável`: não soma ponto e reduz a
   **cobertura**. JA4T ausente em HTTP/3 é inaplicável por estrutura (QUIC não tem SYN).
4. **Famílias causais com teto.** TLS, contagem de cifras, extensões e GREASE vêm do mesmo ClientHello:
   são uma evidência, não quatro. Cada família tem teto positivo e um teto negativo menor (coerência não
   prova humano).
5. **O score não é probabilidade.** Sem rótulos e calibração, é um índice heurístico 0–100. O infopage
   deve escrever "índice não calibrado".
6. **Declarar automação é diferente de esconder.** `python-requests` honesto é automação declarada (papel),
   não fraude. O relatório separa *perfil técnico*, *papel da automação* e *risco de abuso*: o infopage só
   pode medir os dois primeiros (não vê comportamento, rota, conta nem reputação).

## 2. Resultado por parâmetro

Cada regra avaliada produz:

```json
{
  "id": "TLS-GREASE",
  "family": "tls",
  "value": "sem GREASE",
  "expected": "GREASE presente (Chromium, Safari)",
  "state": "contradicao",
  "points": 20,
  "reason": "UA Chrome sem GREASE no ClientHello",
  "benign": "proxy TLS corporativo reescreve o ClientHello"
}
```

| `state` | Cor | Pontos | Significado |
|---|---|---|---|
| `ideal` | verde | ≤ 0 | Igual ao esperado para o cliente declarado. |
| `normal` | azul | 0 | Variação legítima (rede, SO, versão). |
| `atipico` | amarelo | 0 a +10 | Raro para o cliente declarado, com causa benigna comum. |
| `contradicao` | laranja | +10 a +60 | Incompatível com o cliente declarado (+60 só para proxy ou SOCKS confirmado pelo tempo). |
| `declarado` | vermelho | +40 a +60 | O próprio cliente diz ser automação (UA de biblioteca, `webdriver`). |
| `desconhecido`, `nao_aplicavel` | cinza | 0 | Não medido ou sem sentido neste protocolo; reduz a cobertura. |

As regras de contradição **só se aplicam quando o UA declara um navegador** (Chromium, Firefox ou
Safari). Para um UA de biblioteca, a pontuação vem só da declaração.

## 3. Agregação

```text
S_f = clamp(Σ pontos da família f, −teto_neg_f, +teto_pos_f)
S   = Σ S_f
R   = round(100 · σ(0,07 · S − 2,2))       # S = 0 → 10
cobertura = peso das regras avaliadas / peso das regras aplicáveis
```

| Família | Fonte | Teto + | Teto − | Custo de falsificar |
|---|---|---:|---:|---|
| `decl` | UA de automação, `navigator.webdriver` | 60 | 0 | — (é declaração) |
| `tls` | `Ja4`, `Tls`, `Tls-Hello`, `Alpn` | 45 | −15 | médio (uTLS, curl-impersonate) |
| `http` | `Header-Order`, `Ja4h`, `H2`, `H2-Detail`, `H3`, cabeçalhos | 40 | −15 | médio |
| `tcp` | `Ja4t`, `Transport` (SYN, TTL) | 30 | −10 | alto (kernel) |
| `time` | `Timing` | 60 | 0 | alto (física) |
| `js` | APIs do navegador, coletadas pelo `app.js` | 40 | −10 | baixo (controlado pelo cliente) |
| `net` | listas de robôs, nuvens e iCloud Private Relay (`lists.go`) | 45 | 0 | alto (endereço) |

| `R` | Faixa exibida |
|---|---|
| 0–20 | Compatível com navegador |
| 21–50 | Indeterminado |
| 51–80 | Suspeito de automação |
| 81–100 | Automação provável |

- Cobertura < 50% → exibir "evidência insuficiente" no lugar da faixa.
- Para uma futura decisão de bloqueio (fora do infopage): exigir duas famílias acima de 1,5× o peso típico,
  contando **`tls` + `http` como uma só** ("pilha de aplicação"), porque uma biblioteca que imita o TLS
  costuma imitar o HTTP/2 junto (curl_cffi).

## 4. Catálogo de regras

Pesos de partida, adaptados das tabelas Deepseek (§5.2 e §7.2 do anexo) e corrigidos pelas posições
consolidadas (§21 do relatório). Valores de referência vêm dos goldens em
`pkg/fingerprint/testdata/golden/` (Chrome 156, Firefox 159, curl 8, Go 1.27) e precisam ser revistos a
cada versão de navegador.

### 4.1 `decl` — declaração

| ID | Parâmetro | Ideal | Pontos | Falso positivo |
|---|---|---|---:|---|
| DECL-UA | `User-Agent` com `curl`, `python-requests`, `Go-http-client`, `HeadlessChrome`, `PhantomJS`, `Selenium`… | UA de navegador | +60 | Ferramenta legítima (é o papel, não o abuso) |
| DECL-WD | `navigator.webdriver` | `false` | +50 | Testes automatizados do próprio dono |
| DECL-UA (robô) | UA de robô (~40 conhecidos + `crawler`, `spider`, `bot/`) | endereço na lista oficial (+ DNS reverso e direto para Google, Bing, Apple) | 0 se verificado (papel `crawler_verificado`); +60 fora da lista ou com DNS que não confere; +40 sem lista para verificar | Robôs legítimos se declaram |
| DECL-UA (vazio) | `User-Agent` ausente | presente | +40 | — |

A declaração é uma regra só (`DECL-UA`); `navigator.webdriver` (`DECL-WD`, +50) vem do JS e cai na mesma
família. **Automação declarada não recebe alívios**: os pontos negativos de coerência são zerados, porque um
Chrome headless é coerente por construção.

### 4.2 `tls` — ClientHello

| ID | Parâmetro (`X-Traefix-*`) | Ideal para navegador | Fora do ideal | Pontos | Falso positivo |
|---|---|---|---|---:|---|
| TLS-VER | `Tls` `ver` | `1.3` | `1.2` com navegador moderno | +20 | Proxy TLS antigo |
| TLS-CIPHERS | `Ja4` (contagem de cifras, posições 4–5) | 15–20 | > 25 (curl/OpenSSL 30, LibreSSL 49) ou < 15 (Go 13) | +20 | Proxy de inspeção |
| TLS-ALPN | `Alpn` | `h2,http/1.1` | sem `h2`, ou ALPN ausente (`Ja4` termina em `00`) | +30 | Proxy que quebra ALPN |
| TLS-SNI | `Ja4` `d`/`i` | `d` | `i` com navegador | +5 | Acesso direto por IP (legítimo) |
| TLS-GREASE | `Tls-Hello` (`g` nas listas) | Chromium e Safari: presente; Firefox: ausente | contrário do motor declarado | +20 | Proxy TLS |
| TLS-EXT-ENGINE | `Tls-Hello` `exts` | ALPS (`4469`/`44cd`) só Chromium; `record_size_limit` (`001c`) só Firefox; `compress_certificate` (`001b`) em todos | extensão do motor errado, ou falta `001b` | +20 | — (raro) |
| TLS-PQ | `Tls` `group`, `Tls-Hello` `groups` | `X25519MLKEM768` (Chrome ≥ 131, Firefox ≥ 132, Safari 26) | ausente em versão que o tem por padrão | +5 | Política corporativa desliga PQ |
| TLS-ECH | `Tls-Hello` `ech_ext` | Chrome/Firefox: `1` | `0` em Chrome/Firefox | +5 | Política, versão antiga |
| TLS-KNOWN | `Ja4` × clientes medidos (`known_clients.json` do finger-sniffer e goldens) | JA4 do navegador declarado (−15) ou não catalogado (0) | igual ao de biblioteca (+45), de outro navegador (+40), ou cifras na ordem padrão de OpenSSL/LibreSSL/Go/Python (+30) | ±45 | Proxy que termina o TLS |
| TLS-MATCH | demais regras TLS coerentes, sem TLS-KNOWN | — | — | −10 | Clone bem feito (por isso o teto negativo) |

Em QUIC (`q…`) não se aplicam `TLS-CIPHERS`, `TLS-GREASE`, `TLS-EXT`, `TLS-PQ` nem `TLS-ECH`: todos os
clientes medidos oferecem as mesmas 3 cifras do TLS 1.3, e não há referência de extensões por motor. O ALPN
esperado passa a ser `h3`.

`TLS-CIPHERS`, `TLS-GREASE` e `TLS-EXT-ENGINE` costumam disparar juntas para uma biblioteca: o teto de 45 faz
delas uma evidência forte, não três.

### 4.3 `http` — requisição

O infopage tem uma vantagem: os dados vêm da **própria requisição de navegação** (SPEC.md), então o esperado
é conhecido: `Sec-Fetch-Mode: navigate`, `Sec-Fetch-Dest: document`, `Accept` com `text/html`.

| ID | Parâmetro | Ideal para navegador | Fora do ideal | Pontos | Falso positivo |
|---|---|---|---|---:|---|
| HTTP-COUNT | `Header-Order` (quantidade) | ≥ 10 (Chrome 13 numa navegação) | < 8 | +20 | Proxy que remove cabeçalhos |
| HTTP-FETCH | `Sec-Fetch-*` | presentes e `navigate/document` | ausentes (todos os motores enviam desde 2023) ou incoerentes | +25 | Navegador muito antigo |
| HTTP-CH | `Sec-CH-UA*` | Chromium: presentes; Firefox/Safari: ausentes | Chromium sem CH em HTTPS; Firefox/Safari **com** CH | +25 / +30 | Navegador Chromium de nicho |
| HTTP-CH-PLAT | `Sec-CH-UA-Platform` × SO do UA | igual | diferente | +30 | Quase nenhum |
| HTTP-CH-MOB | `Sec-CH-UA-Mobile` × UA | igual | diferente | +10 | Emulação de dispositivo do DevTools |
| HTTP-ENC | `Accept-Encoding` | `gzip, deflate, br, zstd` (Safari sem `zstd`) | sem `br` em HTTPS | +20 | Proxy transparente |
| HTTP-LANG | `Accept-Language` (`Ja4h` posições 9–12) | presente | ausente (`0000`) | +15 | resistFingerprinting do Firefox |
| HTTP-ORDER | `Header-Order` × modelo do motor | ordem relativa do motor (Chrome: `sec-ch-ua…` antes de `user-agent`; Firefox: `user-agent` primeiro) | ordem de outro motor ou de biblioteca | +20 | Extensão que injeta cabeçalhos |
| HTTP-CASE | `Header-Order` em h1 | `Title-Case` | tudo minúsculo em h1 | +10 | Proxy HTTP/1 (não prova automação, §21 item 21) |
| HTTP-H2-PSEUDO | `H2` (4º campo) | Chrome `m,a,s,p`; Firefox `m,p,a,s` | outro (curl `m,s,a,p`, Go `a,m,p,s`) | +30 | — (raro) |
| HTTP-H2-SET | `H2` (SETTINGS e WINDOW_UPDATE) | Chrome `1:65536;2:0;4:6291456;6:262144\|15663105`; Firefox `1:65536;2:0;4:131072;5:16384\|12517377` | de outra família | +20 | Safari/WebKit muda por versão |
| HTTP-PROTO | `Proto` × `Alpn` | `h2` quando ofereceu `h2` | `h1` com `h2` oferecido | 0, `desconhecido` | Investigar o coletor antes (§13) |
| HTTP-MATCH | conjunto coerente com o motor | — | — | −15 | "Alívio" principal, mas sozinho não absolve |

`HTTP-COUNT`, `HTTP-ORDER` e `Ja4h` derivam da mesma lista de cabeçalhos: a quantidade é contada uma vez.

### 4.4 `tcp` — SYN e IP

Só com `fingerprint.tcp` (Linux) e fora do HTTP/3. O que o CRITERIA.md classifica como qualidade (MSS,
escala de janela, SACK, DSCP, reenvios) **não pontua**.

| ID | Parâmetro | Ideal | Fora do ideal | Pontos | Falso positivo |
|---|---|---|---|---:|---|
| TCP-OS | `Ja4t` (ou `syn_opts` + `syn_win`) × SO do UA | Apple `65535_2-1-3-1-1-8-4-0-0`; Linux/Android `2-4-8-1-3`; Windows `2-1-3-1-1-4` (goldens `tcp/`) | pilha de outro SO | −10 / +20 | VPN/relay que termina o TCP; iCloud Private Relay (o endereço na lista zera a regra) |
| TCP-TTL | `ttl_init` × SO do UA (TCP e HTTP/3) | 64 Unix/Apple/Android, 128 Windows | diferente, inclusive 255 | +10 (0 se `TCP-OS` já disparou: mesma causa) | VPN, proxy, roteador |
| TCP-RAW | `syn_opts` só MSS, ou sem opções | opções completas | SYN mínimo de ferramenta de pacote cru | +25 | Equipamento muito antigo |
| TCP-HOPS | `hops` ≤ 2 com UA de consumidor | — | origem na mesma rede/datacenter | +5 (+25 quando houver `net` = hosting) | Rede local, teste do próprio dono |

### 4.5 `time` — temporização da conexão (proxy ou SOCKS)

O handshake TCP termina em quem abriu a conexão; a volta do TLS vai até o navegador. Com um proxy ou SOCKS entre o
cliente e o Traefix:

- a volta TLS (`tls_rtt_us`) passa a volta TCP (`tcp_rtt_us`) pela distância do proxy ao cliente (`rtt_gap_us`);
- o primeiro byte do ClientHello chega uma volta do proxy depois do ACK (`est_to_first_us`), porque o proxy só
  avisa o cliente que o túnel abriu depois do handshake TCP com o servidor. Direto, o ClientHello sai junto com o
  ACK.

| ID | Parâmetro | Ideal | Fora do ideal | Pontos | Falso positivo |
|---|---|---|---|---:|---|
| TIME-RELAY | `Timing` `rtt_gap_us`, `tcp_rtt_us`, `est_to_first_us` | diferença perto de 0, ClientHello em até 5 ms | `rtt_gap_us` > 10 ms **e** > 30% de `tcp_rtt_us` **e** `est_to_first_us` > 5 ms | **+60**, papel `via_proxy`; +5 com `est_to_first_us` ≤ 5 ms (Wi-Fi ou dispositivo lento); 0 no iCloud Private Relay | VPN ou proxy corporativo; pico de Wi-Fi que atrase o TLS e o primeiro byte na mesma conexão |
| TIME-NEG | `rtt_gap_us` / `est_to_first_us` negativos | — | — | 0 | Legítimos pela semântica do coletor (§9.3) |

- **Limiares** medidos pelo finger-sniffer no finger-01 (task 313, `docs/dev/traefix-finger-01.md` daquele
  projeto): SOCKS5 com gap de 5,8–68,7 ms e primeiro byte de 5,1–442 ms; clientes diretos com primeiro byte de
  0,1–4,9 ms. Com `rtt_gap` > 10 ms e `est_to_first` > 5 ms, 43 de 50 conexões pelo SOCKS e 2 de 40 diretas
  (picos de Wi-Fi com os dois sinais juntos).
- **Peso.** Proxy ou SOCKS é tratado como contorno de limites por IP: +60 leva sozinho a `R` = 88 ("Automação
  provável"). Como na automação declarada, **não há alívio**: os pontos negativos de coerência (TLS, HTTP, JS) são
  zerados, porque um navegador real atrás de um proxy é exatamente o caso de quem espalha requisições por
  endereços.
- **Mesma causa.** Com o proxy confirmado, o SYN é do proxy: as regras `tcp` com pontos positivos (TCP-OS,
  TCP-TTL) ficam com 0.
- **HTTP/3** não tem handshake TCP separado do TLS: a regra não se aplica.
- **iCloud Private Relay** (lista `net`) continua sem pontos: é o relay de privacidade do Safari, não um contorno
  escolhido pelo cliente.

Relógio adiantado, latência alta e reenvios são qualidade (CRITERIA.md): `0` ponto.

### 4.6 `js` — navegador

Controlado pelo cliente: barato de falsificar, por isso teto menor que o das camadas de baixo, mas é onde o
navegador real automatizado (Puppeteer, Playwright) deixa rastro.

| ID | Parâmetro | Ideal | Fora do ideal | Pontos | Falso positivo |
|---|---|---|---|---:|---|
| JS-CH-HDR | `userAgentData.brands`/`platform` × `Sec-CH-UA*` do cabeçalho | iguais | diferentes (cabeçalho injetado por ferramenta) | +30 | — |
| JS-UA | `navigator.userAgent` × `User-Agent` do cabeçalho | iguais | diferentes | +30 | Extensão que troca o UA só no cabeçalho |
| JS-PLAT | `navigator.platform` × SO do UA | coerente | diferente | +20 | — |
| JS-GPU | WebGL renderer | GPU real | `SwiftShader`, `llvmpipe` com UA de consumidor | +15 | VM, RDP, GPU desativada |
| JS-LANG | `navigator.languages[0]` × `Accept-Language` | iguais | diferentes | +10 | Raro |
| JS-TOUCH | `maxTouchPoints` × UA móvel | > 0 | 0 com UA móvel | +15 | Emulação do DevTools |
| JS-SCREEN | `screen` | dimensões reais | 0×0 | +20 | — (`outerWidth` 0 ficou de fora: webviews embutidas também o têm) |
| JS-PDF | `pdfViewerEnabled` em Chrome desktop | `true` | `false` | +5 | Política corporativa |
| JS-MISSING | JS não executou | — | — | 0, `desconhecido` | Bloqueador, no-JS (nunca pontuar) |

### 4.7 `net` — endereço

O IP é o de `X-Traefix-Src-Ip` (depois `X-Real-Ip`, `X-Forwarded-For`, o socket).

| ID | Lista | Ideal | Fora do ideal | Pontos | Falso positivo |
|---|---|---|---|---:|---|
| NET-CLOUD | AWS, Google Cloud, Oracle, DigitalOcean, Cloudflare | rede de acesso | datacenter com UA de navegador | +20 (0 para programa declarado ou robô verificado) | VPN, VDI, proxy corporativo |
| NET-RELAY | iCloud Private Relay | — | — | 0, informativo; zera TCP-OS, TCP-TTL e TIME-RELAY | — |
| NET-CRAWLER-IP | listas de robôs | — | endereço de robô com UA de navegador | +10 | Buscador sob demanda do operador |

Listas sem download ainda: `NET-CLOUD` fica `desconhecido`. ASN, Tor e reputação continuam fora
(`tmp/bot-detect-methods.md` §5, candidatos gratuitos: iptoasn, saídas Tor, Spamhaus DROP).

## 5. Exemplos de cálculo

| Cliente | Contribuições | S | R | Faixa |
|---|---|---:|---:|---|
Os cinco primeiros são testes de `score_test.go` com os valores dos goldens.

| Cliente | Contribuições | S | R | Faixa |
|---|---|---:|---:|---|
| Chrome 156 real, direto | TLS-KNOWN −15, HTTP-MATCH −15, TCP-OS −10 | −40 | 1 | Compatível |
| Firefox 159 real | idem | −40 | 1 | Compatível |
| curl com UA de Chrome | TLS teto 45 (conhecido, cifras, GREASE, extensões); HTTP teto 40; TCP −10 | 75 | 95 | Automação provável |
| curl ou Go honestos | DECL-UA +60 (contradições não se aplicam) | 60 | 88 | Automação provável, papel "declarada" |
| Chrome headless | DECL-UA +60, sem alívios; com `webdriver` e SwiftShader, `decl` fica no teto e JS +15 | 60–75 | 88–95 | Automação provável |
| Safari pelo iCloud Private Relay | TCP-OS, TCP-TTL e TIME-RELAY zerados | ≤ 0 | ≤ 10 | Compatível |
| Chrome real por SOCKS5 (task 313) | TIME-RELAY +60; TCP-OS zerado (SYN do proxy); sem alívios | 60 | 88 | Automação provável, papel "por proxy ou SOCKS" |
| Playwright com stealth | só JS-GPU +15; TLS −15; HTTP −15; TCP −10 | −25 | 2 | Compatível |

O último caso é a limitação que o relatório registra: **fingerprint passivo não pega navegador real
automatizado**. Só comportamento, rota e resultado de negócio pegam, e o infopage não os vê.

## 6. Integração no infopage

- **Onde calcula.** Famílias passivas (`decl`, `tls`, `http`, `tcp`, `time`, `net`) em `score.go`, sobre os
  cabeçalhos da requisição: o resultado vai no JSON `infopage-data`, no `/api` (`score`) e na saída de texto
  (`Automation-Score`, `Automation-Rule`). A família `js` e a agregação final no `app.js`, com os tetos e a
  calibração que o servidor envia. Versão das regras: `score/v1`.
- **Aba Resumo.** Cartão "Pontuação de automação" logo após o destaque: índice, faixa, papel, cobertura, TLS
  reconhecido, relay/datacenter/robô; tabela com todas as regras avaliadas por família (valor, esperado, estado,
  pontos e explicação benigna), a soma bruta e a soma com teto de cada família, o total `S`, o índice `R` com a
  fórmula e a cobertura; por fim a "Avaliação completa": faixa, papel, itens com e sem dado, regras que pesaram
  para automação e para navegador, e as famílias limitadas pelo teto.
- **Aba Técnico.** "Regras da pontuação": todas as regras com valor, esperado, estado e pontos.
- **CRITERIA.md** continua sendo o critério de qualidade; este documento, o de automação.

## 7. Cuidados

- **Oráculo de evasão.** Mostrar regras e pesos ao visitante é aceitável no infopage, que é vitrine e
  ferramenta de teste. Um scorer de produção (por exemplo atrás do `forwardAuth`) não devolve razões ao
  cliente.
- **Nada de bloqueio a partir deste índice.** Sem rótulos, calibração e avaliação por coorte (§20 do
  relatório), os pesos são pontos de partida. O `fingerprintList` não aplica score.
- **Revisão por versão.** Chrome muda TLS e HTTP/2 a cada poucos meses; refazer os goldens e a tabela de
  motores a cada versão estável e registrar a data.
- **Futuro `net`.** ASN/hosting, Tor, faixas de crawlers e `rtt × geo` exigem enriquecimento local (MMDB)
  que o Traefix não faz hoje; é a família de maior ganho que falta (ranking §4.8 do anexo Deepseek).
- **Propagação.** Novos cabeçalhos `X-Traefix-*` que virarem regra entram nesta tabela, além do
  CRITERIA.md e da SPEC.md.
