# Imagem tmsoftbrasil/traefix-infopage (linux/amd64 + linux/arm64), compilada de forma cruzada.
FROM --platform=$BUILDPLATFORM golang:1-alpine AS builder

RUN apk --no-cache --no-progress add ca-certificates tzdata \
    && update-ca-certificates

WORKDIR /go/infopage

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o infopage .

FROM scratch

ARG VERSION=dev
ARG REVISION=
ARG CREATED=
LABEL org.opencontainers.image.title="traefix-infopage" \
      org.opencontainers.image.description="Página que mostra as informações do navegador e os cabeçalhos do Traefix (derivada de traefik/whoami)" \
      org.opencontainers.image.source="https://github.com/patrickbrandao/traefix" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$REVISION" \
      org.opencontainers.image.created="$CREATED"

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/infopage/infopage /infopage
COPY LICENSE NOTICE /

ENTRYPOINT ["/infopage"]
EXPOSE 80
