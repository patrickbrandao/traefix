# Proposta ao quic-go: ordem dos SETTINGS no evento de qlog

Rascunho para o usuário enviar ao `quic-go/quic-go` (issue e depois PR). Reescrever com a própria voz antes de
enviar.

## Título

http3/qlog: keep the order of the settings in SettingsFrame

## Motivo

`SettingsFrame` keeps the settings it does not know in `Other map[uint64]uint64`, which loses the order of the
wire. The qlog HTTP/3 schema describes the settings as a list, and the order a client sends them in, GREASE
included, is part of what tells HTTP/3 clients apart (the HTTP/2 counterpart is in the Akamai fingerprint).
Today, getting that order needs a fork or decrypting the 1-RTT packets.

## Mudança

- `qlog.SettingsFrame` gets `Order []uint64`, the identifiers in the received order, filled by
  `parseSettingsFrame` only when a qlog recorder is set.
- `SettingsFrame.encode` writes the settings in that order when it is set, the current order otherwise.
- Tests: the order of a frame with GREASE and known settings, the JSON in wire order.

Implementação: commit "http3: keep the order of the SETTINGS in the qlog event (Traefix)" do branch
`traefix` de `github.com/patrickbrandao/traefix-quic-go` (base v0.65.0).
