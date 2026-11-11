# Plan: Built-in TLS certificates for the NATS listeners

**Branch:** `cbrake/master` **Branched from:** `d8ef593b`

Issue: [#877](https://github.com/simpleiot/simpleiot/issues/877). Related:
[#72](https://github.com/simpleiot/simpleiot/issues/72),
[#812](https://github.com/simpleiot/simpleiot/issues/812).

## Context

A cloud instance runs behind a web proxy such as Caddy, which terminates HTTPS
for browsers and proxies plain HTTP to the Simple IoT HTTP port. That stays.
The certificate this plan is about is the one the NATS, MQTT, and NATS
WebSocket listeners present to devices and MQTT clients that connect to those
ports directly, which the proxy does not front.

Today that certificate comes from two file paths read once at start-up, and in
practice the paths point at the files Caddy writes. That has two problems:
Caddy's key is readable only by the `caddy` user, so Simple IoT has to run as
root or a copy step has to move the files, and a renewed certificate is not
served until the process restarts (#812).

What exists today that the plan builds on:

- **NATS listener TLS** from `SIOT_NATS_TLS_CERT` and `SIOT_NATS_TLS_KEY`
  (`server/args.go`), read once into a static `tls.Config` by
  `server.GenTLSConfig` (`server/nats-server.go`). The same config serves the
  MQTT and NATS WebSocket listeners. `SIOT_NATS_TLS_TIMEOUT` sets the handshake
  timeout for slow devices.
- **Two pinned copies of the certificate.** The server's loopback NATS client
  (`server/server.go`) and the HTTP port's WebSocket proxy (`api/ws-proxy.go`)
  each read the certificate file once and accept exactly that leaf through
  `client.PinnedTLSConfig` (`client/tls.go`), because the certificate carries
  the public name and the connection is to `127.0.0.1`. Both go stale at
  renewal.
- **The HTTP server** (`api/server.go`) serves plain HTTP on 4223 through
  `NewAppHandler`, where a route for the challenge path can be added.
- **The sync client** (`client/edge.go`) trusts the system roots for `tls` and
  `wss` upstreams, with an optional `caCert` point for a private CA, so a
  publicly issued certificate needs no change on devices.
- `golang.org/x/crypto` is already a dependency, and its `acme/autocert`
  package is an ACME client with a certificate cache, background renewal, and
  a handler for the HTTP-01 challenge.
- `server/tls_test.go` generates a self-signed certificate and starts a server
  with TLS, so there is a fixture to build on.

Out of scope: HTTPS on the HTTP port, binding 80 or 443, DNS-01 and wildcard
certificates, OCSP stapling, client certificates, and changes to how devices
trust an upstream.

## Goals

1. An instance with a public name obtains a certificate for its NATS, MQTT,
   and WebSocket listeners from Let's Encrypt, stores it under `SIOT_DATA`, and
   renews it before expiry, with one setting and no change to the proxy.
2. A renewed certificate takes effect without a restart, whether it came from
   ACME or from files that another tool renews (closes #812).
3. Simple IoT owns its own key, so it does not need to read the proxy's files
   or run as root.
4. A certificate from files keeps working, including the `mkcert` development
   flow.
5. Docs, changelog, and `siot_test` green at every phase.

## Design Decisions

### The challenge travels through the proxy

Let's Encrypt validates HTTP-01 by fetching
`http://<domain>/.well-known/acme-challenge/<token>` on port 80, and follows
redirects to port 80 or 443 only. With the usual Caddyfile,

```
siot.example.com {
    reverse_proxy localhost:4223
}
```

Caddy redirects that request to HTTPS and proxies it to the HTTP port like any
other, where a handler on the challenge path returns the token. Caddy's own
ACME handler on port 80 answers only tokens from its own outstanding orders
and passes the rest down the chain, so the two clients do not interfere; each
holds its own certificate for the same name, which is within Let's Encrypt's
limits. Simple IoT binds nothing new.

TLS-ALPN-01 cannot work behind the proxy, since Caddy answers on 443 with its
own certificate. `autocert` tries TLS-ALPN-01 first and, when that validation
fails, places a new order and uses HTTP-01, and it has no option to skip the
first attempt. The cost is one failed validation per order, every 60 days or
so, well inside the CA's limit of five failures per hour. The start-up log
notes this so the failure in the CA's view is not mistaken for a problem.

### `autocert` rather than `certmagic`

`golang.org/x/crypto/acme/autocert` is already in the module graph, adds no
dependency, and covers ordering, caching, and renewal. `certmagic` adds DNS
challenges and OCSP stapling at the cost of a sizable dependency tree for a
binary that also runs on embedded devices.

### One certificate source behind every listener

A new file, `server/certs.go`, defines the one place a certificate comes from:

```go
// certSource hands the current certificate to the NATS listeners and to the
// server's own connections back to those listeners.
type certSource interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
	// Leaf is the DER of the certificate GetCertificate returns right now.
	Leaf() ([]byte, error)
}
```

Two implementations:

- **`fileCerts`** loads a certificate and key pair from paths, or finds the
  pair in a directory (see Settings), and reloads the pair when either file
  changes, comparing size and modification time and
  checking at most once a minute so a busy listener does not stat on every
  handshake. A copied file may keep its original timestamp, so a change in
  either value counts.
- **`acmeCerts`** wraps an `autocert.Manager` with a `DirCache` at
  `<SIOT_DATA>/certs`, `HostWhitelist(domain)`, the contact email, and the CA
  directory URL. `autocert` refuses a hello with no server name, and the
  loopback client and WebSocket proxy dial `127.0.0.1` without one, so the
  wrapper fills in the configured domain when the hello has no SNI or names
  something else. At start-up the wrapper asks for the domain's certificate
  once, so the first order happens immediately rather than on the first
  device connection, and an order failure is in the log before anything
  connects.

`serverTLSConfig(src)` builds the listener config: `MinVersion` 1.2 and
`GetCertificate` from the source. The NATS server takes it as
`opts.TLSConfig` for its client, MQTT, and WebSocket listeners; `GenTLSConfig`
and the `TLSCert`/`TLSKey` option fields go away.

### The loopback connections follow the source instead of pinning a file

`clientTLSConfig(src)` replaces the two `PinnedTLSConfig` calls. It verifies
the presented leaf against `src.Leaf()` at handshake time rather than against a
copy read at start-up, so a renewal is accepted on the next connection and the
server does not have to trust a CA or the name `localhost`.
`client.PinnedTLSConfig` had no other caller and is removed;
`client.LeafCert` is reused by `fileCerts`.

### Settings

The existing names stay, since the certificate still serves only the NATS
family of listeners. Three settings are added:

| Setting                 | Meaning                                                                                                   |
| ----------------------- | --------------------------------------------------------------------------------------------------------- |
| `SIOT_NATS_TLS_DOMAIN`  | Host name to obtain a certificate for. Setting it turns ACME on.                                          |
| `SIOT_NATS_TLS_EMAIL`   | Contact address registered with the CA (optional).                                                        |
| `SIOT_NATS_TLS_CA`      | ACME directory URL. Default is Let's Encrypt production; the staging URL is useful while setting a host up. |
| `SIOT_NATS_TLS_DIR`     | Directory holding a certificate and key managed outside Simple IoT, as an alternative to the two paths. |
| `SIOT_NATS_TLS_CERT`    | Unchanged: certificate chain file, when the certificate is managed outside Simple IoT.                    |
| `SIOT_NATS_TLS_KEY`     | Unchanged: key file.                                                                                      |
| `SIOT_NATS_TLS_TIMEOUT` | Unchanged.                                                                                                |

Only one of the domain, the directory, and the file pair may be set; anything
else is an error at start-up, named in the log.

`SIOT_NATS_TLS_DIR` names a directory and Simple IoT finds the pair in it,
searching subdirectories too. A certificate is a file named `*.crt` or
`fullchain.pem` with a key beside it named `*.key` or `privkey.pem`; when
several pairs are found, the one whose certificate was issued most recently is
used. That covers the layout Caddy writes, where each issuer has its own
directory under `certificates/`:

```
certificates/acme-v02.api.letsencrypt.org-directory/<domain>/<domain>.crt
certificates/acme.zerossl.com-v2-dv90/<domain>/<domain>.crt
```

Pointing the setting at `certificates/` follows Caddy whichever issuer it used,
including its fallback from Let's Encrypt to ZeroSSL, on a host where Caddy
holds one domain. A host where Caddy holds several points at the domain's
directory under one issuer, or at the ACME setting instead. certbot's
`live/<domain>/` works the same way. No pair at all is an error naming the
directory. The search runs again on each reload check, so a renewal that
writes a new pair elsewhere in the tree is picked up as soon as its
certificate is the newest.

### The challenge route

`NewAppHandler` mounts `autocert.Manager.HTTPHandler` at
`/.well-known/acme-challenge/` when ACME is on. The handler is given a fallback
that returns 404, so the route never redirects; the proxy owns redirects. The
route is reachable without authentication, which is what the challenge needs,
and it serves only tokens the manager placed, so it exposes nothing.

### Development

`siot_mkcert` and `siot_run_tls` in `envsetup.sh` are unchanged. For ACME
there is no local flow; the manual test uses Let's Encrypt staging on a host
with a public name behind Caddy.

## Changes to note in the changelog

- A renewed certificate file is served without a restart.
- New: `SIOT_NATS_TLS_DIR` names a directory holding the certificate and key,
  in place of the two file paths.
- New: `SIOT_NATS_TLS_DOMAIN`, `SIOT_NATS_TLS_EMAIL`, `SIOT_NATS_TLS_CA`, and
  the challenge route on the HTTP port.
- No incompatible changes.

## Phases

### Phase 1: One certificate source, reloaded on change

- [x] `server/certs.go`: `certSource`, `fileCerts` with reload on change and
      the directory scan, `serverTLSConfig`, `clientTLSConfig`.
- [x] `server/args.go`: `SIOT_NATS_TLS_DIR`; the error when it is set with the
      file pair.
- [x] `server/nats-server.go`: `natsServerOptions` takes `TLSConfig *tls.Config`
      in place of `TLSCert`/`TLSKey`; `GenTLSConfig` is no longer called.
- [x] `server/server.go`: build the source from the options once, pass the
      server config to the NATS server and the client config to the loopback
      connection; drop the `os.ReadFile` of the certificate.
- [x] `api/ws-proxy.go` and `api.ServerArgs`: take the client `*tls.Config`
      rather than a certificate path.
- [x] Tests in `server/certs_test.go`: `fileCerts` serves the new pair after the
      files are rewritten, including a rewrite that keeps the modification
      time; the directory search finds a Caddy layout and a certbot layout,
      picks the newest of two issuers' pairs, refuses a directory with none,
      and follows a renewal that writes a new pair under another issuer; `clientTLSConfig` accepts the renewed leaf and
      refuses a different one. In `server/tls_test.go`, the existing cases pass with the new
      options, plus one that rewrites the certificate while the server runs and
      reconnects the loopback client and the WebSocket proxy.
- [x] `docs/user/configuration.md` documents `SIOT_NATS_TLS_DIR` and notes
      that a changed certificate file is picked up without a restart; changelog
      entry. Close #812.
- [x] `siot_test` passes. Commit.

### Phase 2: ACME

- [x] `server/acme.go`: `acmeCerts` around `autocert.Manager` with the
      `DirCache` under `SIOT_DATA`, host whitelist, email, CA URL, the SNI
      fill-in, and the start-up request for the domain's certificate.
- [x] `server/args.go`: the three settings; the error for domain plus files.
- [x] `api/server.go`: the challenge route in `NewAppHandler`, passed the
      manager's handler through `ServerArgs`.
- [x] Start-up log line naming the domain, the CA, and the cache directory. A
      failed order is logged with the CA's reason, since the common failures
      (DNS not pointing here, the proxy not passing the path through, rate
      limited) are all diagnosed from that message. The log also notes that the
      CA will record one failed TLS-ALPN-01 attempt per order.
- [x] Tests: the SNI fill-in returns the domain's certificate for an empty and
      for a foreign server name, against a `Manager` whose cache is pre-seeded
      with a self-signed certificate so no CA is contacted; the challenge route
      returns a token the manager placed and 404 otherwise; the settings parse;
      domain plus files is refused. An opt-in test behind `SIOT_TEST_ACME_DIR`
      runs a full order against a local
      [pebble](https://github.com/letsencrypt/pebble) when the variable names
      its directory URL, with pebble's HTTP-01 port pointed at the test server.
- [ ] Manual: a scratch host with a public name behind the Caddyfile above,
      `SIOT_NATS_TLS_DOMAIN` set and `SIOT_NATS_TLS_CA` set to the Let's
      Encrypt staging URL. Caddy's access log shows the challenge request
      arriving on 80, redirected, and proxied to 4223 with a 200. A `tls` NATS
      upstream, a `wss` upstream (which sees Caddy's certificate), and an MQTT
      client all connect; the cache directory survives a restart without a new
      order.
- [x] `siot_test` passes. Commit.

### Phase 3: Docs and changelog

- [x] `docs/user/installation.md`: the cloud section keeps Caddy for HTTPS and
      adds the one setting for the NATS certificate, with the note that the
      proxy must pass the challenge path through, which the default Caddyfile
      does.
- [x] `docs/user/configuration.md`: the three settings in the NATS block.
- [x] `docs/ref/security.md`: the deployment checklist's NATS port item names
      `SIOT_NATS_TLS_DOMAIN` as the way to get a certificate; the HTTP section
      mentions the unauthenticated challenge route and what it serves.
- [x] `docs/user/sync.md` and `docs/user/mqtt.md`: point at the setting.
- [x] `CHANGELOG.md`; `plans/plans.md`. Close #877 and #72.
- [x] `siot_test` passes. Commit.

## Testing Strategy

- **Unit (Go):** `fileCerts` reload, the client config following a renewal,
  the SNI fill-in, the challenge route, settings parsing.
- **Integration (Go):** the existing TLS server tests with the new options and
  a renewal while running.
- **Opt-in (Go):** the full order against pebble when `SIOT_TEST_ACME_DIR` is
  set.
- **Manual:** the staging CA on a public host behind Caddy, as listed in
  Phase 2. This is the check on the one behavior taken from certmagic's source
  rather than observed: that Caddy's challenge handler passes a foreign token
  through.

## Open Questions

- Whether the failed TLS-ALPN-01 attempt before each HTTP-01 order is
  acceptable long term. The alternative is driving `golang.org/x/crypto/acme`
  directly, which is a few hundred lines of order handling that `autocert`
  already has. Start with `autocert` and revisit if the CA's failure limit is
  ever a problem.
- Whether `SIOT_NATS_TLS_DIR` pointed at Caddy's directory is worth
  documenting, given that reading it needs root or a copy step. It is the
  layout the setting is shaped for, so the docs should show it with that
  caveat, and point at `SIOT_NATS_TLS_DOMAIN` as the way to avoid it.
- Whether the challenge route should be mounted always rather than only with
  ACME on. It serves 404 without a manager, so mounting it always is harmless;
  decide while writing the handler.

## Follow-ups (not in this plan)

- HTTPS on the HTTP port and direct challenge answering on 80 and 443, for a
  deployment with no proxy. The source, the route, and the settings here are
  the first half of that; it needs the port binding, a redirect listener, and a
  capability line in the service unit.
- A dedicated `siot` service user in `siot install`. Nothing here needs root,
  since Simple IoT owns its key under `SIOT_DATA`.
- DNS-01 for a host whose port 80 is not reachable at all.
