# Development

## Go Package Documentation

The Simple IoT source code is
[available](https://github.com/simpleiot/simpleiot) on Github.

Simple IoT is written in Go.
[Go package documentation](https://pkg.go.dev/github.com/simpleiot/simpleiot) is
available.

## Code Organization

Currently, there are a lot of subdirectories. One reason for this is to limit
the size of application binaries when building edge/embedded Linux binaries. In
some use cases, we want to deploy app updates over cellular networks, therefore
we want to keep packages as small as possible. For instance, if we put the
`natsserver` stuff in the `nats` package, then app binaries grow a couple MB,
even if you don't start a NATS server. It is not clear yet what Go does for dead
code elimination, but at this point, it seems referencing a package increases
the binary size, even if you don't use anything in it. (Clarification welcome!)

For edge applications on Embedded Linux, we'd eventually like to get rid of
net/http, since we can do all network communications over NATS. We're not there
yet, but be careful about pulling in dependencies that require net/http into the
NATS package, and other low level packages intended for use on devices.

### Directories

See Go docs
[directory descriptions](https://pkg.go.dev/github.com/simpleiot/simpleiot#section-directories)

## Coding Standards

Please run `siot_test` from `envsetup.sh` before submitting pull requests. All
code should be formatted and linted before committing.

Please configure your editor to run code formatters:

- **Go**: `goimports`
- **Elm**: `elm-format`
- **Markdown**: `prettier` (note, there is a `.prettierrc` in this project that
  configures prettier to wrap markdown to 80 characters. Whether to wrap
  markdown or not is debatable, as wrapping can make diffs harder to read, but
  Markdown is much more pleasant to read in an editor if it is wrapped. Since
  more people will be reading documentation than reviewing, lets optimize for
  the reading in all scenarios -- editor, Github, and generated docs)

## Pure Go

We plan to keep the main Simple IoT application a pure Go binary if possible.
Statically linked pure Go has huge advantages:

1. you can easily cross compile to any target from any build machine.
2. blazing fast compile times
3. deployment is dead simple – zero dependencies. Docker is not needed.
4. you are not vulnerable to security issues in the host systems SSL/TLS libs.
   What you deploy is pretty much what you get.
5. although there is high quality code written in C/C++, it is much easier to
   write safe, reliable programs in Go, so I think long term there is much less
   risk using a Go implementation of about anything – especially if it is widely
   used.
6. Go’s network programming model is much simpler than about anything else.
   Simplicity == less bugs.

Once you link to C libs in your Go program, you forgo many of the benefits of
Go. The Go authors made a brilliant choice when they chose to build Go from the
ground up. Yes, you loose the ability to easily use some of the popular C
libraries, but what you gain is many times more valuable.

## Building Simple IoT

Requirements:

- Go
- Node/NPM

Simple IoT build has currently been testing on Linux and MacOS systems. See
[`envsetup.sh`](https://github.com/simpleiot/simpleiot/blob/master/envsetup.sh)
for scripts used in building.

To build:

- `source envsetup.sh`
- `siot_setup`
- `siot_build`

## Running unit tests

There are not a lot of unit tests in the project yet, but below are some
examples of running tests:

- test everything: `go test -race ./...`
- test only client directory: `go test -race ./client`
- run only a specific: `go test -race ./client -run BackoffTest (run takes a
  RegEx)
- `siot_test` runs tests as well as vet/lint, frontend tests, etc.

The leading `./` is important, otherwise Go things you are giving it a package
name, not a directory. The `...` tells Go to recursively test all subdirs.

## Document and test during development

It is much more pleasant to write documentation and tests as you develop, rather
than after the fact. These efforts add value to your development if done
concurrently. Quality needs to be
[designed-in](https://community.tmpdir.org/t/podcast-280-cristiano-amon-qualcomm-ceo-lex-fridman-podcast/515),
and
[leading with documentation](https://handbook.tmpdir.org/documentation/lead-with-documentation.html)
will result in
[better thinking](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/07/leslie_lamport.pdf)
and a better product.

If you develop a feature, please update/create any needed documentation and
write any tests (especially end-to-end) to verify the feature works and
continues to work.
