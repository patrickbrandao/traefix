# Add a ServersTransport option to prefer IPv6 for backend connections

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13417](https://github.com/traefik/traefik/pull/13417) |
| Tipo | Recurso |
| Área | service/serversTransport |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | RamazanKara |
| Aberto em | 2026-06-29 (última atualização: 2026-10-18) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +157/-4 em 17 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/service |
| Status | adiado |

## O que faz

Adiciona a opção preferIPv6 ao ServersTransport. Com ela, o hostname do backend é resolvido para IPv6 primeiro e cai para IPv4 só se nenhum IPv6 for alcançável. Literais de IP e conexões com família já fixada não mudam. Vale para o transport dinâmico (file, labels, KV), o CRD e o default@internal. Closes #10677.

## Riscos e situação no upstream

Atinge 17 arquivos, com CRD gerado e applyconfiguration. Status 2-needs-review, sem review; autor diz que roda com patch no próprio setup. Aplica no dialer, então h2c explícito não é afetado. Útil em dual-stack com custo de egress IPv4.

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/config/static/static_config.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/serverstransportspec.go`
- `docs/content/reference/routing-configuration/http/load-balancing/serverstransport.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13417/head:upstream-pr-13417`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.**

Achados no PR original:

- O fallback é sequencial e sem Happy Eyeballs. `preferIPv6DialContext` tenta `tcp6` e só tenta `tcp4` quando o primeiro falha. Quando o AAAA resolve mas o IPv6 está em blackhole (pacotes descartados sem RST, situação comum em dual-stack mal configurado), cada conexão nova espera o `dialTimeout` inteiro (padrão de 30s) antes de cair para IPv4. Com `preferIPv6: true`, isso degrada justamente o caso em que a opção deveria ajudar. O correto seria uma corrida estilo RFC 8305: IPv6 primeiro e IPv4 depois de ~300 ms, com o vencedor fechando o perdedor.
- Ele faz duas resoluções DNS por conexão quando há fallback (uma para `tcp6`, outra para `tcp4`).
- O `net.Dialer` do Go já faz Happy Eyeballs com `"tcp"`, ordenando os endereços pela RFC 6724, que normalmente já prefere IPv6 quando há rota global. O ganho real se limita a ambientes em que a ordenação do resolvedor coloca IPv4 primeiro.
- Não se aplica a backends h2c nem a serviços TCP (que usam `pkg/tcp/dialer.go`), o que é inconsistente para uma opção de `ServersTransport`.
- O PR traz CRD e `applyconfiguration` gerados. Seria preciso regenerar com `script/code-gen.sh`, que neste ambiente (macOS) não roda sem ajuste (`sed -i` no formato GNU).
- O upstream ainda está em "need-triage", sem review.

Reavaliar se o upstream aceitar o design. Se for implementar localmente, fazer a corrida IPv6/IPv4 com atraso de fallback e cobrir também o dialer TCP.

Nenhum código importado e nenhum teste rodado.
