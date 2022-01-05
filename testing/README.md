# Testing

This directory contains misc testing scripts, tools, etc.

## Running Caddy on development machine

This can be used to test TLS support for various things like wss:// upstream
URIs, etc.

- configure one SIOT instance (the upstream) to run on port 8081
- `cd <this directory>`
- install caddy and mkcert (Arch has packages for both)
- `mkcert -install`
- `mkcert siottest.com`
- sudo vi /etc/hosts
  - `127.0.0.1 siottest.com`
- `sudo caddy run -watch`
- start another SIOT instance (the downstream) and configure upstream with URL
  set to wss://siottest.com -- it will not connect to SIOT instance running on
  port 8081.
