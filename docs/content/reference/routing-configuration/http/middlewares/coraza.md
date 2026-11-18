---
title: "Traefik Coraza Documentation"
description: "Inspect requests and responses with the OWASP Coraza WAF and the OWASP Core Rule Set compiled into Traefix, with the coraza middleware."
---

The `coraza` middleware inspects the requests and the responses with the [OWASP Coraza](https://coraza.io/) web application firewall, which runs [SecLang](https://coraza.io/docs/seclang/) directives compatible with ModSecurity. Coraza 3.8.1 and the [OWASP Core Rule Set](https://coreruleset.org/) 4.25.0 are compiled into the binary: nothing is downloaded at start, and no WASM runtime runs. The middleware is specific to Traefix, the fork of Traefik Proxy.

## Configuration Examples

```yaml tab="Structured (YAML)"
# Detects the attacks with the Core Rule Set, without blocking them.
http:
  middlewares:
    waf:
      coraza:
        crsEnabled: true
        directives:
          - Include @coraza.conf-recommended
          - Include @crs-setup.conf.example
          - Include @owasp_crs/*.conf
          - SecRuleEngine DetectionOnly
          - SecRequestBodyAccess On
```

```toml tab="Structured (TOML)"
# Detects the attacks with the Core Rule Set, without blocking them.
[http.middlewares.waf.coraza]
  crsEnabled = true
  directives = [
    "Include @coraza.conf-recommended",
    "Include @crs-setup.conf.example",
    "Include @owasp_crs/*.conf",
    "SecRuleEngine DetectionOnly",
    "SecRequestBodyAccess On",
  ]
```

```yaml tab="Labels"
# Detects the attacks with the Core Rule Set, without blocking them.
# The labels use the plugin form, whose indexed list keeps the commas of the SecRule directives.
labels:
  - "traefik.http.middlewares.waf.plugin.coraza.crsEnabled=true"
  - "traefik.http.middlewares.waf.plugin.coraza.directives[0]=Include @coraza.conf-recommended"
  - "traefik.http.middlewares.waf.plugin.coraza.directives[1]=Include @crs-setup.conf.example"
  - "traefik.http.middlewares.waf.plugin.coraza.directives[2]=Include @owasp_crs/*.conf"
  - "traefik.http.middlewares.waf.plugin.coraza.directives[3]=SecRuleEngine DetectionOnly"
  - "traefik.http.middlewares.waf.plugin.coraza.directives[4]=SecRequestBodyAccess On"
```

```yaml tab="Kubernetes"
# The plugin form of Traefik Hub, which the Middleware resource accepts.
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: waf
spec:
  plugin:
    coraza:
      crsEnabled: true
      directives:
        - Include @coraza.conf-recommended
        - Include @crs-setup.conf.example
        - Include @owasp_crs/*.conf
        - SecRuleEngine DetectionOnly
        - SecRequestBodyAccess On
```

### Plugin Form

Traefix reads `plugin.coraza`, the form of Traefik Hub and of the Coraza WASM plugin, as this middleware when the install configuration declares no plugin named `coraza`. In labels, use the plugin form with an indexed list, `directives[0]`, `directives[1]` and so on: the `coraza.directives` label splits its value on commas, which a `SecRule` directive contains. To move from the WASM plugin, remove its `experimental.plugins.coraza` block: the routing configuration stays the same. In the Kubernetes `Middleware` resource, the plugin form is the only one available.

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-directives" href="#opt-directives" title="#opt-directives">`directives`</a> | SecLang directives of the WAF, one per entry, for example `SecRuleEngine On`. `Include` reads the files of the host from their path, for example `Include /etc/traefix/waf/*.conf`, and the embedded Core Rule Set from the names starting with `@`. An invalid directive, or an `Include` of a missing file, disables the routers using the middleware, whose clients get a `404`, as with any invalid middleware. | | Yes |
| <a id="opt-crsEnabled" href="#opt-crsEnabled" title="#opt-crsEnabled">`crsEnabled`</a> | Makes the embedded Core Rule Set available to `Include`: `@coraza.conf-recommended`, `@crs-setup.conf.example` and `@owasp_crs/*.conf`. When unset, an `Include` of these names disables the router. | `false` | No |
| <a id="opt-failOpen" href="#opt-failOpen" title="#opt-failOpen">`failOpen`</a> | Lets the request through when the WAF fails to process it, for example when the request body can't be read. When unset, Traefix answers `500` to such requests. Both cases count as `error` in the metrics. | `false` | No |

Traefix reads the rule files when it builds the middleware. A change in a file applies at the next change of the routing configuration that rebuilds the middleware.

## Roll Out the Core Rule Set

1. Start with `SecRuleEngine DetectionOnly`. The requests that the rules would block go through and count as `detected` in `traefik_waf_requests_total`.
2. Look at the matched rules in `traefik_waf_rule_matches_total` and in the audit log. To write the audit log to the standard output, add `SecAuditEngine RelevantOnly`, `SecAuditLog /dev/stdout` and `SecAuditLogFormat JSON`.
3. Exclude the false positives: `SecRuleRemoveById 942100` after the `Include` of the rules, or `ctl:ruleRemoveById` in a rule before them, for a path.
4. Switch to `SecRuleEngine On`. The requests whose anomaly score reaches the threshold of rule `900110` get a `403`.

## Disk Use

Coraza keeps the request body in memory up to `SecRequestBodyInMemoryLimit`, and writes the larger bodies to the temporary directory, `/tmp` in the Traefix image, which declares it as a volume. With a read-only root file system, mount a writable volume on `/tmp`, or set `SecRequestBodyInMemoryLimit` to the value of `SecRequestBodyLimit` to keep the bodies in memory. The persistent collections of `SecDataDir` also need a writable directory.

## Response Inspection

With `SecResponseBodyAccess On`, and a response whose content type is listed in `SecResponseBodyMimeType`, Traefix holds the response back until the WAF lets it through, up to `SecResponseBodyLimit`. A flush of the backend then waits for the verdict. A blocked response loses its headers and body, and gets the status code of the rule. A connection taken over, such as a WebSocket, is no longer inspected.

## Fingerprints in Rules

The rules can read the headers of the [entry point fingerprint](../../../install-configuration/entrypoints-fingerprint.md), which the entry point sets before any middleware, for example:

```text
SecRule REQUEST_HEADERS:X-Traefix-Ja4 "@beginsWith t13d4907h2_" "id:10001,phase:1,deny,status:403,msg:'curl client'"
```

## Metrics

| Metric | Labels | Contains |
|:-------|:-------|:---------|
| <a id="opt-traefik-waf-requests-total" href="#opt-traefik-waf-requests-total" title="#opt-traefik-waf-requests-total">`traefik_waf_requests_total`</a> | `middleware`, `result` | Requests processed: `allowed`, `detected` (a rule with the `log` action matched without blocking), `blocked`, `error`. |
| <a id="opt-traefik-waf-duration-seconds" href="#opt-traefik-waf-duration-seconds" title="#opt-traefik-waf-duration-seconds">`traefik_waf_duration_seconds`</a> | `middleware` | Time spent in the WAF, without the next handlers. |
| <a id="opt-traefik-waf-rule-matches-total" href="#opt-traefik-waf-rule-matches-total" title="#opt-traefik-waf-rule-matches-total">`traefik_waf_rule_matches_total`</a> | `middleware`, `rule_id` | Matches of the rules with the `log` action. The setup rules of the Core Rule Set are `nolog` and don't count. |

Datadog, StatsD and InfluxDB use the same names with dots, for example `traefik.waf.requests.total`. Traefix also logs each logged rule match at the `DEBUG` level.
