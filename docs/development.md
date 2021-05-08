+++
title = "Development"
weight = 3
+++

## Code Organization

Currently, there are a lot of subdirectories. One reason for this is to limit
the size of application binaries when building edge/embedded Linux binaries. In
some use cases, we want to deploy app updates over cellular networks, therefore
we want to keep packages as small as possible. For instance, if we put the
`natsserver` stuff in the `nats` package, then app binaries grow a couple MB,
even if you don't start a nats server. It is not clear yet what Go does for dead
code elimination, but at this point, it seems referencing a package increases
the binary size, even if you don't use anything in it. (Clarification welcome!)

For edge applications on Embedded Linux, we'd eventually like to get rid of
net/http, since we can do all network communications over NATS. We're not there
yet, but be careful about pulling in dependencies that require net/http into the
nats package, and other low level packages intended for use on devices.

### Directories

See https://pkg.go.dev/github.com/simpleiot/simpleiot#section-directories

## Coding Standards

Please run `siot_test` from `envsetup.sh` before submitting pull requests. All
code should be formatted and linted before committing.

Please configure your editor to run code formatters:

- Go: `goimports`
- Elm: `elm-format`
- Markdown: `prettier` (note, there is a `.prettierrc` in this project that
  configures prettier to wrap markdown to 80 characters. Whether to wrap
  markdown or not is debatable, as wrapping can make diff's harder to read, but
  Markdown is much more pleasant to read in an editor if it is wrapped. Since
  more people will be reading documentation than reviewing, lets optimize for
  the reading in all scenarios -- editor, Github, and generated docs)

* [Environment Variables](environment-variables.md)

## Running unit tests

There are not a lot of unit tests in the project yet, but below are some
examples of running tests:

- test everything: `go test ./...`
- test only db directory: `go test ./db`

The leading `./` is important, otherwise Go things you are giving it a package
name, not a directory. The `...` tells Go to recursively test all subdirs.
