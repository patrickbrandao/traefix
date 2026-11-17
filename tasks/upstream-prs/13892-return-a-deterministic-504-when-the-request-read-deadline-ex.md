# Return a deterministic 504 when the request read deadline expires

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13892](https://github.com/traefik/traefik/pull/13892) |
| Tipo | Correção |
| Área | server/entrypoint-timeouts |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | abrahamgracef |
| Aberto em | 2026-09-12 (última atualização: 2026-09-14) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +194/-3 em 5 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/server |
| Status | adiado |

## O que faz

Torna determinístico o status quando respondingTimeouts.readTimeout estoura durante o upload do body: passa a ser sempre 504 (e não alternar com 499), registrando o deadline no contexto da requisição. Um desconexão real do cliente antes do deadline continua como 499.

## Riscos e situação no upstream

Mexe em ComputeStatusCode e no contexto da requisição no entrypoint. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/proxy/httputil/observability.go`
- `pkg/proxy/httputil/proxy.go`
- `pkg/server/server_entrypoint_tcp.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13892/head:upstream-pr-13892`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, status "review". Revisores externos apontaram uma corrida no próprio mecanismo, sem decisão de maintainer.

### Achados no PR original

- **Heurística com corrida reconhecida pelo autor:** o PR distingue timeout de leitura de desconexão do cliente comparando
  `time.Now()` com um deadline recalculado na entrada do handler, que não é o deadline realmente aplicado à conexão.
  Uma desconexão real perto do deadline vira `504`, e a corrida entre os dois caminhos de cancelamento só fica menor,
  não desaparece.
- **Mudança de API exportada:** `ComputeStatusCode(err)` vira `ComputeStatusCode(ctx, err)` e a chave de contexto é do
  tipo string (`key = "ReadDeadline"`), contra a convenção do projeto de usar chaves struct não exportadas.
- O ganho é só cosmético (o status registrado em access log e métricas, `499` ou `504`), sem impacto de segurança.
- Sobreposição com o #13528 (já importado), que mexe no mesmo `ComputeStatusCode` e adiciona um teste
  `Test_ComputeStatusCode`. O Traefix também tem um `withReadTimeout` local no HTTP/3 que precisaria ser considerado.

### Motivo do adiamento

Benefício pequeno contra uma mudança de assinatura exportada e uma heurística com corrida conhecida. Vale reavaliar se o
upstream aceitar a abordagem ou passar a usar o deadline real da conexão.

### Testes

Nenhum. O PR não foi aplicado.
