# Critérios de avaliação TCP/IP do infopage

Base do cartão "Avaliação TCP/IP" e dos "Sinais" da aba Resumo (`static/assets/app.js`, funções
`evaluate` e `signals`). Pesquisa de 2026-11-19 (RFCs, código do kernel Linux, p0f, FoxIO JA4T, Cloudflare
AIM, ITU-T, Anatel).

**Princípio:** quase nada no SYN e no `TCP_INFO` de uma conexão recém-aberta diz se a conexão é boa ou
ruim; a maior parte descreve o sistema operacional ou a rede de acesso. Cores:

| Cor | Uso |
|---|---|
| Verde (`ok`) | Valor que efetivamente ajuda o usuário (latência baixa, SACK, escala de janela). |
| Azul (`info`) | Característica do sistema ou da rede, sem valor "melhor" (timestamps, ECN, MSS de PPPoE, saltos, reenvios). |
| Amarelo (`warn`) | Incoerência ou anomalia (TTL × User-Agent, DF desligado, opções IP, sem SACK, sem escala de janela). |
| Vermelho (`bad`) | Só o que o usuário sente com certeza (automação declarada, sem criptografia, relógio > 5 min). |

## Camada IP

| Valor | Ideal | Classificação | Por quê |
|---|---|---|---|
| `ttl`, `hops` | — | azul | Saltos não medem qualidade (MPLS esconde saltos, VPN encurta). RFC 791/8200. |
| `ttl_init` × User-Agent | igual | verde / amarelo se diferente | 64 = Linux, Android, macOS, iOS; 128 = Windows; 255 = equipamento. Diferente = proxy, VPN, Private Relay ou reescrita de TTL; nunca fraude por si só (p0f). |
| `dscp` | 0 | azul | Remarcado na maioria dos caminhos da internet (RFC 2474, 8325). |
| `ecn` (IP do SYN) | 0 | azul; CE (3) amarelo | RFC 3168 §6.1.1; ECT no SYN é experimento (RFC 8311). |
| `df` (IPv4) | 1 | amarelo se 0 | Todo SO moderno liga DF (RFC 1191). |
| `ipid` | qualquer | azul | Com DF o ID não tem significado (RFC 6864); Linux manda ID ≠ 0 no SYN (`id+` no p0f). |
| `ipopt` | 0 | amarelo se > 0 | Navegadores não usam opções IP (RFC 7126). |
| `flow_label` | ≠ 0 | azul | Só identifica a pilha (RFC 6437). |
| `syn_mss` | 1460 (v4) / 1440 (v6) | vira o MTU estimado (abaixo) | PPPoE 1452/1432, WireGuard 1380/1360, Tailscale 1240/1220, móvel 1360–1460. Impacto real de 1% a 3% na vazão (APNIC 2019). |
| MTU estimado | 1500 | régua de MTU (abaixo) | MTU para o qual o cliente dimensionou os pacotes; não entra na pontuação de automação. |
| `pmtu` | — | azul | MTU da rota do servidor até o cliente, não o do cliente. Um blackhole de PMTUD não aparece no SYN (RFC 2923). |

## MTU estimado

O cartão "MTU estimado" da aba Resumo mostra o MTU para o qual o cliente dimensionou os pacotes, somando os
cabeçalhos ao que ele anunciou ao abrir a conexão:

| Conexão | Base | Transporte | IP | Exemplo |
|---|---|---:|---:|---|
| TCP | `syn_mss` (opção MSS do SYN) | TCP 20 | IPv4 20, IPv6 40 | 1460 + 20 + 20 = 1500; 1440 + 20 + 40 = 1500 |
| HTTP/3 | `max_udp_payload` (parâmetro de transporte do QUIC, em `X-Traefix-H3` `tp=`) | UDP 8 | IPv4 20, IPv6 40 | Chrome 1472 + 8 + 20 = 1500; quic-go 1452 + 8 + 20 = 1480 |

- Igual ao `mtu_est` do `X-Traefix-Transport` no TCP (MSS + 40 ou + 60).
- Sem `max_udp_payload`, ou com o padrão 65527 do RFC 9000 (curl, Firefox), o cliente não planejou um MTU: o
  cartão diz "sem plano" e mostra só o tamanho do primeiro datagrama Initial, que é fixo do cliente (Chrome 1250,
  curl 1200) e não do caminho.
- Sem `X-Traefix-H3` com `tp=` (o Traefix não leu o ClientHello do QUIC nesta conexão, por exemplo numa conexão
  retomada), o cartão diz que não há dado, sem culpar o cliente.
- **Régua** (nota de 0 a 100, interpolação linear entre os pontos):

| MTU | Nota | Leitura |
|---:|---:|---|
| ≥ 1500 | 100 | Ethernet/Wi-Fi sem túnel |
| 1492 | 95 | PPPoE (fibra e DSL) |
| 1480 | 90 | PPPoE com VLAN, túnel leve |
| 1420 | 51 | WireGuard |
| 1400 | 39 | IPsec, rede móvel |
| ≤ 1340 | 0 | Mínimo crítico: pacotes grandes podem se perder |

  Cores: verde ≥ 95, azul 60–94, amarelo 30–59, vermelho < 30. A nota mede a rede de acesso (túneis,
  encapsulamento), não a vazão, e **não entra na pontuação de automação**.

## Opções TCP do SYN

| Valor | Classificação | Por quê |
|---|---|---|
| `syn_win` | azul | Arredondado para múltiplo do MSS (64240 = 44×1460) ou fixo 65535 na Apple; nunca escalado (RFC 7323 §2.2). |
| `syn_wscale` | verde ≥ 5, azul 1–4, amarelo 0/ausente | Sem escala a janela fica em 64 KB. 65535 × 2^ws é teto teórico: a janela real vem do autotuning. |
| `syn_sack` | verde; amarelo se ausente | RFC 2018. |
| `syn_ts` | azul | Windows desliga por padrão. |
| `syn_ecn` | azul | Apple pede; Linux (`tcp_ecn=2`) e Windows não. |
| `syn_tfo` | azul | RFC 7413; raro na web. |
| `syn_opts` | azul | Assinatura do SO (JA4T). |

`tcp_info` do Linux (lado do servidor, conferido no código do kernel):

- `mss_snd` (`tcpi_snd_mss`) = min(MSS do cliente, PMTU − 40) − 12 com timestamps.
- `mss_adv` (`tcpi_advmss`) também desconta os 12 bytes após o handshake (`tcp_input.c`): 1448/1448 é
  coerente. É dado do **servidor**.
- `mss_rcv` (`tcpi_rcv_mss`) começa em 536 e só sobe com segmentos grandes: 536 é normal.
- `wscale_rcv`, `cc`, `cwnd` são do servidor: azul ou omitidos.

## Métricas da conexão

| Valor | Classificação | Por quê |
|---|---|---|
| Latência (`rtt_min_us`, depois `tcp_rtt_us`, `rtt_syn_us`, `quic_min_rtt_us`) | verde < 80 ms; azul 80–300 ms; amarelo > 300 ms | Depende da distância (≈ 1 ms por 100 km de fibra). Cloudflare AIM; Anatel ≤ 80 ms terrestre, ≤ 900 ms satélite. Não usar `rtt_us` (srtt): inclui a verificação do certificado pelo cliente. |
| `rttvar_us` | não exibir como jitter | O kernel o inicia em RTT/2 (RFC 6298); jitter exige várias medições no navegador. |
| `retrans` | azul | Conexão curta não mede perda; 1 reenvio pode ser sonda TLP (RFC 8985). |
| `lost` | não usar | `tcpi_lost` é instantâneo, não acumulado. |
| `rtt_gap_us`, `est_to_first_us` | vermelho "proxy ou SOCKS" se `rtt_gap_us` > 10 ms, > 30% de `tcp_rtt_us` **e** `est_to_first_us` > 5 ms; amarelo "indício fraco" só com o gap | Os dois juntos separam proxy de Wi-Fi lento (task 313 do finger-sniffer: 43 de 50 conexões por SOCKS, 2 de 40 diretas). Detalhes e peso na pontuação: SCORE.md §4.5. |
| Relógio | amarelo > 60 s; vermelho > 5 min | TOTP e Kerberos falham a partir de minutos; certificados a partir de 1 dia. |

## Erros que o infopage não deve cometer

1. Nota de MTU como perda de vazão (1448 contra 1460 é 0,8%, não 3%): a régua de MTU mede o encapsulamento do caminho.
2. MSS abaixo de 1460 em vermelho.
3. Timestamps, ECN ou TFO ausentes como defeito.
4. `rttvar` chamado de jitter.
5. "RWIN máximo limita a X Mbps" como fato.
6. Retransmissões ou perdas de uma conexão curta como qualidade.
7. Dados do servidor (`advmss`, `rcv_mss`, `wscale_rcv`, `cc`, `pmtu`) apresentados como do cliente.

## Fontes principais

- RFCs 791, 1191, 2018, 2474, 2923, 3168, 6298, 6437, 6864, 6928, 7126, 7323, 7413, 8200, 8311, 8325, 8985.
- Kernel Linux: `net/ipv4/tcp.c` (`tcp_get_info`), `tcp_input.c` (`tcp_rtt_estimator`,
  `tcp_initialize_rcv_mss`, desconto de `TCPOLEN_TSTAMP_ALIGNED` no `advmss`), `tcp_output.c`
  (`tcp_mtu_to_mss`, `tcp_select_initial_window`).
- p0f (`p0f.fp`, README); FoxIO JA4T; goldens em `pkg/fingerprint/testdata/golden/tcp/`.
- APNIC, "TCP MSS values – what's changed?" (2019); Cloudflare AIM (`ScoresCalculations.ts`);
  Anatel, indicadores SCM; ITU-T G.114, G.1010, Y.1541.
