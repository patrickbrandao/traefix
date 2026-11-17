# Add CRL-based client certificate revocation support for mTLS

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14014](https://github.com/traefik/traefik/pull/14014) |
| Tipo | Recurso |
| Área | tls/mtls, provider/kubernetes-crd |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | vantrop |
| Aberto em | 2026-10-06 (última atualização: 2026-10-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +5186/-31 em 42 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/tls, area/provider/k8s/crd |
| Status | adiado |

## O que faz

Adiciona o tipo de autenticacao RequireAndVerifyClientCertWithExpiry, que checa revogacao via CRL de cada certificado da cadeia do cliente. As CRLs vem de um store global em arquivo (tls.crlFiles) ou da CDP do certificado, com configuracao por TLSOption.

## Riscos e situação no upstream

Muito grande (+5186/-31, 42 arquivos, inclui CRD gerado e fsnotify). Label status/2-needs-review; seis reviews COMMENTED e nenhum aprovado. Pontos de seguranca: CRLs do store global sao confiadas implicitamente, sem verificar assinatura do emissor nem thisUpdate/nextUpdate. Exige revisao cuidadosa antes de portar.

## Principais arquivos

- `cmd/traefik/traefik.go`
- `pkg/config/static/static_config.go`
- `pkg/api/handler_certificate_test.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/clientauth.go`
- `docs/content/reference/install-configuration/tls/crl.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14014/head:upstream-pr-14014`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.**

Revisei manualmente as partes centrais (`pkg/tls/crl*.go`, `tlsmanager.go`, `tls.go`, `cmd/traefik/traefik.go`,
`static_config.go`). O código gerado do CRD e os testes eu só percorri por alto.

Achados (PR original):
- **CRLs do store global confiadas implicitamente.** O próprio PR diz que as CRLs de `tls.crlFiles` não têm a
  assinatura verificada contra o emissor e que `thisUpdate`/`nextUpdate` não são checados. Nesse caminho, uma
  CRL expirada, truncada ou de outra CA é aceita como fonte de verdade. Além disso, o store global **tem
  precedência** sobre o HTTP e **ignora a allowlist**. Na prática, uma CRL vazia ou velha esconde
  revogações sem nenhum aviso.
- **Provável panic na inicialização.** `cmd/traefik/traefik.go` lê `staticConfiguration.TLS.CRLFiles`, mas
  `TLS` é um ponteiro (`*TLSClientAuth`) que fica nil quando a seção `tls` não está na configuração de
  instalação, e o diff não inicializa esse default. Quem não usa CRL derrubaria o processo no boot.
- **Download da CDP dentro do handshake.** A busca HTTP da CRL acontece em `tls.Config.VerifyConnection`, de
  forma síncrona, com timeout padrão de 30 s. Com `FailedClosed`, os handshakes concorrentes ficam bloqueados
  durante o refresh. Um cliente com certificado válido pode prender goroutines e conexões até o timeout, e
  um CDP lento ou fora do ar degrada todo o entry point mTLS.
- **SSRF e memória com a allowlist desligada por padrão** (`whitelist.enabled: false`). Qualquer URL de CDP
  presente num certificado da cadeia verificada é buscada com um clone do `http.DefaultTransport`, que
  respeita `HTTP(S)_PROXY` e não restringe destinos internos. O cache por distribution point não tem limite
  de entradas (o próprio comentário admite "high memory usage"). O risco é menor porque a CDP vem de um
  certificado assinado pela CA confiável, mas uma CA intermediária comprometida ou mal configurada vira
  vetor de SSRF interno.
- `fetchCRL` aloca `maxCRLBytes+1` antes de ler (1 MiB por padrão, configurável sem teto) em cada download,
  mesmo quando a CRL é pequena. Pontos positivos: há `LimitReader`, os redirects estão desabilitados e o
  caminho HTTP verifica assinatura, issuer, freshness e monotonicidade do `CRL Number`.
- Escopo e API: o PR cria um novo `clientAuthType` (`RequireAndVerifyClientCertWithExpiry`) em vez de uma opção
  ortogonal, uma seção `tls` nova na configuração de instalação, campos novos no CRD `TLSOption`, código
  gerado e +5186 linhas. No upstream há seis reviews `COMMENTED`, nenhuma aprovação e nenhuma decisão de
  maintainer sobre a API. O review automatizado apontou `new(time.Duration(...))`: isso compila no Go do
  `go.mod` (o Go 1.26 aceita `new(expr)`), mas mostra que a revisão está incompleta.
- Não encontrei código suspeito, como telemetria ou chamadas de rede fora do download da CRL.

Motivo do adiamento: para importar com segurança, seria preciso, no mínimo:
1. verificar a assinatura das CRLs do store global contra a CA confiável do TLSOption (`clientAuth.caFiles`) ou
   contra o emissor da cadeia verificada, e rejeitar CRL expirada (`nextUpdate` vencido) ou com `thisUpdate`
   no futuro, ou seja, aplicar ao arquivo as mesmas checagens do caminho HTTP;
2. corrigir o nil pointer de `staticConfiguration.TLS`;
3. tirar o download do caminho do handshake (refresh em background, com handshake usando só o cache, e
   política de fail-closed explícita quando não houver CRL válida), aplicar um timeout curto e limitar a
   concorrência por CDP;
4. tornar a allowlist obrigatória para o modo HTTP (ou o default), restringir o esquema a `http`/`https`,
   limitar o número de entradas do cache e impor um teto para `maxCRLBytes`;
5. revisar a API junto com o upstream (`clientAuthType` novo vs opção ortogonal), já que ela vira contrato
   público no CRD.

Isso é praticamente uma reescrita do núcleo (~1.400 linhas fora testes e código gerado), numa área em que um
erro enfraquece o mTLS. O risco é alto demais para um import sem o design aprovado no upstream. Reavaliar
quando houver review de maintainer. Até lá, a revogação pode ser feita com a CA intermediária e a rotação
de certificados de vida curta.

Testes: nenhum (não importado).
