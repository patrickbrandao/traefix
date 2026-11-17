# Reuse cached plugin archive when plugins.traefik.io is unreachable

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13006](https://github.com/traefik/traefik/pull/13006) |
| Tipo | Correção |
| Área | plugins |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | 5queezer |
| Aberto em | 2026-04-19 (última atualização: 2026-11-01) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +105/-4 em 2 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/M, area/plugins |
| Status | importado com ajustes |

## O que faz

Quando o download de um plugin falha e já existe um archive em cache para a mesma versão, usa o cache em vez de dar erro. Hoje a falha de rede no boot faz manager.ResetAll() derrubar todos os middlewares de plugin, e a correção mantém o hash pinning obrigatório.

## Riscos e situação no upstream

Base v3.7 (autor fez rebase para v3.7.17), então precisa adaptar para master. Um maintainer respondeu que o comportamento atual é escolha de design intencional, então a aceitação não está garantida. Comentários de vários usuários relatam outages reais após reboot; não há revisão formal.

## Principais arquivos

- `pkg/plugins/manager.go`
- `pkg/plugins/manager_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13006/head:upstream-pr-13006`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, em "design-review" desde maio sem decisão de maintainer; vários usuários afetados.

### Achados no PR original

- **Archive carregado sem verificação de integridade:** no fallback, qualquer `Check` com falha era tolerado, sob a premissa
  de que o archive em disco "foi validado na instalação anterior" — nada garantia isso. O arquivo podia ser:
  - um download parcial: o `Download` escrevia direto no caminho final com `os.Create`, então uma queda no meio do corpo
    deixava um archive truncado e devolvia erro, disparando o fallback;
  - um archive que nunca passou na validação, ou alterado depois dela.
- **Rejeição explícita do registry ignorada:** o fallback disparava em qualquer erro de `Download`, inclusive `404`
  (versão removida/revogada), e tolerava `Check` respondendo "inválido" com o registry acessível.
- Corpo de erro do registry lido sem limite (`io.ReadAll`).

### Ajustes (reescrita do fallback)

- Ao concluir uma verificação com sucesso (hash fixado ou `Check` do registry), o hash é gravado em `<versão>.zip.verified`.
- O fallback exige que o hash do archive em cache bata com o `hash` configurado ou com o hash verificado gravado
  (comparação em tempo constante). Sem registro, ou com divergência, a instalação falha como antes.
- O fallback só ocorre quando o registry está indisponível (erro de rede/transporte, `5xx`, `429`). Respostas `4xx`
  continuam fatais. No fallback, o `Check` ainda é tentado e uma rejeição explícita (`4xx`) é fatal.
- `Download` grava num arquivo temporário e faz `rename`, então um download interrompido nunca substitui o archive válido.
- Erros de status do registry viraram um tipo (`registryStatusError`); corpo de erro limitado a 4 KiB.
- `CleanArchives` remove também o `.verified` das versões antigas.
- Doc: seção "Registry Unavailability" em `experimental/plugins.md`.

### Testes

- Novos casos: archive verificado reaproveitado (erro de rede e `502`/`503`), archive nunca verificado, archive alterado
  depois da verificação, sem archive, `404` no download sem fallback, `4xx` no `Check` durante o fallback, hash fixado
  divergente, gravação do `.verified`, download interrompido preservando o archive, erros tipados.
- `go build ./...` ok; `go test ./pkg/plugins/` ok; `golangci-lint run ./pkg/plugins/...`: 0 issues.
