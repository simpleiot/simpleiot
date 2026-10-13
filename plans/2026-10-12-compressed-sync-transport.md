# Plan: Compressed Sync Transport

**Branch:** `cbrake/master` **Branched from:** `527d555a`

## Context

A device's sync traffic is dominated by repetition. Each point publish is about
260 bytes, almost all of it a subject, reply inbox, and stream name that repeat
from one message to the next. Replaying a recorded sync stream through
compressors that flush after every write measured this (see the Compression idea
in [Bandwidth](../docs/ref/bandwidth.md#compression)):

| Compression                         | Bytes, both directions | Saving |
| ----------------------------------- | ---------------------- | ------ |
| None                                | 30,921                 |        |
| Deflate, each message on its own    | 24,654                 | 20%    |
| Deflate, state kept across messages | 3,601                  | 88%    |
| Same, with a 4 KB window            | 3,792                  | 88%    |
| zstd, state kept across messages    | 3,736                  | 88%    |

The per-message row matches WebSocket compression as the NATS client and the
HTTP port's proxy implement it (19% measured), since both compress each message
on its own. Neither nats.go nor nats-server offers compression with shared state
on a client connection, so Simple IoT provides the transport itself. Sync, the
store, subjects, and IDs are unchanged.

## Approach

```
device                                      upstream
nats.go ─► zconn ─► TLS ─► counter ─► TCP ═══► listener: TLS ─► zconn ─► InProcessConn ─► nats-server
```

- **`zconn`** wraps a `net.Conn`: each `Write` is compressed with raw deflate
  (`klauspost/compress/flate`, already a dependency through nats-server) and
  flushed with a sync flush, so no message waits; `Read` decompresses. Deflate
  and zstd measured the same, and deflate is the smaller, simpler stream.
- **Device side:** two URI schemes, `natsz://` and `tlsz://`, mirroring
  `nats://` and `tls://`. `EdgeConnect` maps them to `nats://host:port` and
  installs a dialer that layers TLS (for `tlsz://`) and `zconn` over the
  counting connection, so `txBytes`/`rxBytes` count what crosses the network.
  nats.go itself does no TLS on these connections.
- **Upstream side:** a listener on its own port that accepts TCP, terminates TLS
  when the server has a certificate, unwraps `zconn`, and pipes the stream into
  `Server.InProcessConn()`. The server marks in-process connections as not
  requiring TLS, so nats.go and the server agree. This follows the pattern of
  `api/ws-proxy.go`.
- TLS sits outside the compression on both ends, since compressed ciphertext
  saves nothing.

## Decisions

- **Port:** default `SIOT_NATS_PORT + 3` (4225), set with `--natszPort` /
  `SIOT_NATSZ_PORT`, `0` to turn it off. The listener applies the same
  authorizer as the NATS port, so it opens no new access.
- **Window:** 4 KB to start. It kept the full saving in the replay and keeps
  memory per connection small on a hub with many devices. Revisit with a larger
  recording.
- **Stream header:** each direction starts with a short magic and version
  (`SIOTZ1`), so a `natsz://` client pointed at a plain NATS port, or the
  reverse, fails with a clear error and the format can change later.

## Phases

### Phase 1: `zconn`

- [ ] `client/zconn.go`: `net.Conn` wrapper with the stream header, deflate per
      write with a sync flush, and streaming decompression on read. Safe for one
      reader and one writer at a time, which is how nats.go uses a connection.
- [ ] Keep the CONNECT exchange out of the compression history. Compressing a
      secret alongside data an attacker can influence can leak it through
      message sizes (the CRIME class of attacks); the shared token travels in
      CONNECT. Either start a fresh deflate stream after the first write, or
      send the first write uncompressed. Device credentials sign a one-time
      nonce and carry no reusable secret.
- [ ] Tests: round trip of many small writes, partial reads, large writes, close
      and error propagation, a bad header refused, and a ratio check on a
      synthetic stream of repeated point publishes.
- [ ] `go test -race ./client/...`, `golangci-lint run`. Commit.

### Phase 2: Upstream listener

- [ ] Server option and argument for the port (`server/args.go`,
      `server/server.go`), started with the NATS server and stopped with it.
- [ ] Listener: accept, TLS with the server's certificate when it has one,
      unwrap `zconn`, `InProcessConn()`, copy both ways, close both sides
      together.
- [ ] Every connection arrives in-process, so check each place that relies on a
      client's address and give it the real remote address or an equivalent:
  - [ ] the shared token rule under `SIOT_DEVICE_AUTH=required` (the WebSocket
        proxy applies it before forwarding with `connectHasToken`; do the same
        here on the first write);
  - [ ] the authorizer and any sign-in rate limiting keyed by address;
  - [ ] connection logging and `connz`, so an operator can still tell devices
        apart.
- [ ] Tests (`server/`): a device credential connects over the listener and is
      scoped as on the NATS port; a token from a remote address is refused when
      device auth is required; TLS is required when the server has a
      certificate.
- [ ] `go test -race ./server/...`, `golangci-lint run`. Commit.

### Phase 3: Device side

- [ ] `client/uri.go`: accept `natsz://` and `tlsz://`, default port 4225, and
      report which transport was chosen.
- [ ] `client/edge.go`: dialer that layers counting, TLS (with `CACert` when
      set), and `zconn`; leave `nats.Secure` off for these schemes.
- [ ] Set `nats.IgnoreDiscoveredServers()` for these schemes, so a reconnect
      never moves the device to a plain NATS URL the server advertises.
- [ ] Sync UI: the URI placeholder lists the new schemes.
- [ ] Tests (`client/sync_test.go`): `TestSync` and `TestSyncOfflineCatchup`
      over `natsz://`; a sync over `tlsz://` with a test certificate.
- [ ] `go test -race ./client/...`, `golangci-lint run`, `elm-review`. Commit.

### Phase 4: Measure and document

- [ ] Measure a scratch device and upstream over `natsz://` against `nats://`
      with the same load, counting at the device's socket, and record the result
      on the bandwidth page.
- [ ] Docs:
  - [ ] `docs/ref/bandwidth.md`: move compression from Ideas into the main text
        with the measured numbers.
  - [ ] `docs/user/sync.md`: the new schemes, the port, and when to use them.
  - [ ] `docs/ref/security.md`: the new listener goes through the same
        authorizer, and how the token rule applies to it.
  - [ ] `CLAUDE.md`: the authentication paragraph lists the new listener.
  - [ ] `CHANGELOG.md`: a new way to connect a device that uses far less data.
- [ ] Commit.
- [ ] Field check: point `edge-bw-test` at `tlsz://` once its upstream runs this
      release, and compare the `edge-sync` dashboard over a day.

## Incompatibilities for the changelog

- None for existing deployments: `nats://`, `tls://`, `ws://`, and `wss://` keep
  working. A device using `natsz://` or `tlsz://` needs an upstream on this
  release, and the upstream needs the new port reachable through any firewall.
