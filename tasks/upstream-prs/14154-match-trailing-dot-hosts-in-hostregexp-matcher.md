# Match trailing-dot hosts in HostRegexp matcher

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14154](https://github.com/traefik/traefik/pull/14154) |
| Tipo | Correção |
| Área | rules/muxer |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-10-28 (última atualização: 2026-10-28) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +29/-2 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/rules |
| Status | importado |

## O que faz

HostRegexp passa a repetir o match sem o ponto final quando o host da requisição termina com ponto, como o Host já faz. Antes, app.localhost. não casava com HostRegexp(^app\.localhost$) e pulava os middlewares do router. A tentativa com o host bruto continua primeiro.

## Riscos e situação no upstream

Mudança pequena em pkg/muxer/http/matcher.go, com testes. Status: needs-review, sem docs. Baixo risco de regressão.

## Principais arquivos

- `pkg/muxer/http/matcher.go`
- `pkg/muxer/http/matcher_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14154/head:upstream-pr-14154`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, só com review automatizado (sem achados), status "review".

### Achados no PR original

- **Correção relevante para segurança:** `Host` já aceitava o host com ponto final (`app.example.com.`), mas `HostRegexp`
  não. Um request para `app.example.com.` deixava de casar com o router `HostRegexp` e caía num router de prioridade
  menor, pulando os middlewares daquele router (auth, allowlist...). É um bypass de roteamento.
- A correção tenta primeiro o host bruto (regras que incluem o ponto continuam funcionando) e depois o host sem o ponto
  final, como o matcher `Host`.
- Limitação, igual à do `Host`: só um ponto final é removido (`example.com..` não casa), o que é seguro porque falha fechado.
- Fora do escopo: o `HostRegexp` da sintaxe v2 (`matcher_v2.go`, baseado no gorilla/mux) não foi alterado e pode ter a
  mesma limitação. Vale uma task própria se a sintaxe v2 estiver em uso.

### Ajustes

Nenhum.

### Testes

- `go test ./pkg/muxer/...` ok; `golangci-lint run ./pkg/muxer/...`: 0 issues.
