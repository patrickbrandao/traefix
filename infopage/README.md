# infopage

Traefix Infopage: a [whoami](https://github.com/traefik/whoami) fork that also serves, to browsers, a page
showing everything the browser and its connection reveal, with the headers added by Traefix
(`X-Traefix-*`). Image: `tmsoftbrasil/traefix-infopage`. Specification (pt-BR): [SPEC.md](SPEC.md).

Tiny Go webserver that prints OS information and HTTP request to output.

## Usage

### Paths

#### `/[?wait=d][&format=html|text]`

Returns the infopage information (request and network information).

A `GET` whose `Accept` header contains `text/html` (a browser) gets the HTML page, with the "Resumo" and
"Técnico" tabs; other clients, such as curl, get the text output. `format=html` or `format=text` forces
one of them.

#### `/assets/*`

Files of the HTML page (`app.js`, `style.css`).

The optional `wait` query parameter can be provided to tell the server to wait before sending the response.
The duration is expected in Go's [`time.Duration`](https://golang.org/pkg/time/#ParseDuration) format (e.g. `/?wait=100ms` to wait 100 milliseconds).

The optional `env` query parameter can be set to `true` to add the environment variables to the response.

#### `/api`

Returns the infopage information (and some extra information) as JSON.

The optional `env` query parameter can be set to `true` to add the environment variables to the response.

#### `/bench`

Always return the same response (`1`).

#### `/data?size=n[&unit=u]`

Creates a response with a size `n`.

The unit of measure, if specified, accepts the following values: `KB`, `MB`, `GB`, `TB` (optional, default: bytes).

#### `/echo`

WebSocket echo.

#### `/health`

Heath check.

- `GET`, `HEAD`, ...: returns a response with the status code defined by the `POST`
- `POST`: changes the status code of the `GET` (`HEAD`, ...) response.

### Flags

| Flag      | Env var              | Description                             |
|-----------|----------------------|-----------------------------------------|
| `cert`    |                      | Give me a certificate.                  |
| `key`     |                      | Give me a key.                          |
| `cacert`  |                      | Give me a CA chain, enforces mutual TLS |
| `port`    | `INFOPAGE_PORT_NUMBER` | Give me a port number. (default: `80`)  |
| `name`    | `INFOPAGE_NAME`        | Give me a name.                         |
| `prefix`  | `INFOPAGE_HEADER_PREFIX` | Prefix of the Traefix headers, the `fingerprint.headerPrefix` of the entry point. (default: `X-Traefix-`) |
| `lists`   | `INFOPAGE_LISTS`       | Download the address lists of robots, clouds and iCloud Private Relay for the automation score; `0` turns it off. (default: on) |
| `verbose` |                      | Enable verbose logging.                 |

## Examples

```console
$ docker run -d -p 8080:80 --name iamfoo tmsoftbrasil/traefix-infopage

$ curl http://localhost:8080
Hostname: 9c9c93da54b5
IP: 127.0.0.1
IP: ::1
IP: 172.17.0.2
RemoteAddr: 172.17.0.1:41040
GET / HTTP/1.1
Host: localhost:8080
User-Agent: curl/8.5.0
Accept: */*
```

```console
# updates health check status
$ curl -X POST -d '500' http://localhost:8080/health

# calls the health check
$ curl -v http://localhost:8080/health
* Host localhost:8080 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:8080...
* Connected to localhost (::1) port 8080
> GET /health HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/1.1 500 Internal Server Error
< Date: Fri, 18 Apr 2025 13:36:02 GMT
< Content-Length: 0
```

```console
$ openssl req -newkey rsa:4096 \
    -x509 \
    -sha256 \
    -days 3650 \
    -nodes \
    -out ./certs/example.crt \
    -keyout ./certs/example.key

$ docker run -d -p 8080:80 -v ./certs:/certs --name iamfoo tmsoftbrasil/traefix-infopage --cert /certs/example.crt --key /certs/example.key

$ curl https://localhost:8080 -k --cert certs/example.crt  --key certs/example.key
Hostname: 25bc0df47b95
IP: 127.0.0.1
IP: ::1
IP: 172.17.0.2
RemoteAddr: 172.17.0.1:50278
Certificate[0] Subject: CN=traefik.io,O=TraefikLabs,L=Lyon,ST=France,C=FR
GET / HTTP/1.1
Host: localhost:8080
User-Agent: curl/8.5.0
Accept: */*
```

```console
$ docker run -d -p 8080:80 --name iamfoo tmsoftbrasil/traefix-infopage

$ grpcurl -plaintext -proto grpc.proto localhost:8080 infopage.Infopage/Infopage
{
  "hostname": "5a45e21984b4",
  "iface": [
    "127.0.0.1",
    "::1",
    "172.17.0.2"
  ]
}

$ grpcurl -plaintext -proto grpc.proto localhost:8080 infopage.Infopage/Bench
{
  "data": 1
}
```

```yml
version: '3.9'

services:
  infopage:
    image: tmsoftbrasil/traefix-infopage
    command:
       # It tells infopage to start listening on 2001 instead of 80
       - --port=2001
       - --name=iamfoo
```