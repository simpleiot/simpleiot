# Testing

This directory contains misc testing scripts, tools, etc.

## Running Caddy on development machine

- `cd <this directory>`
- install caddy and mkcert (Arch has packages for both)
- `mkcert -install`
- `mkcert siottest.com`
- sudo vi /etc/hosts
  - `127.0.0.1 siottest.com`
- `sudo caddy run -watch`
