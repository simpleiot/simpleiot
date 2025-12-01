# Architecture Decision Records

This directory is used to capture Simple IoT architecture and design decisions.

For background on ADRs see
[Documenting Architecture](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
Decisions by Michael Nygard. Also see
[an example](https://github.com/nats-io/nats-architecture-and-design) of them
being used in the NATS project. The
[Go proposal process](https://github.com/golang/proposal#readme) is also a good
reference.

## Process

When thinking about architectural changes, we should
[lead with documentation](https://handbook.tmpdir.org/documentation/lead-with-documentation/).
This means we should start a branch, draft a ADR, and then open a PR. An
associated issue may also be created.

ADRs should used primarily when a number of approaches need to be considered,
thought through, and we need a record of how and why the decision was made. If
the task is a fairly straightforward implementation, write documentation in the
existing User and Reference Guide sections.

When an ADR is accepted and implemented, a summary should typically be added to
the [Reference Guide](../../README.md) documentation.

See [template.md](template.md) for a template to get started.

## ADRs

| Index                                           | Description                                 |
| ----------------------------------------------- | ------------------------------------------- |
| [ADR-1](1-consider-changing-point-data-type.md) | Consider changing/expanding point data type |
| [ADR-2](2-authz.md)                             | Authorization considerations.               |
| [ADR-3](3-node-lifecycle.md)                    | Node lifecycle                              |
| [ADR-4](4-time.md)                              | Notes on storing and transferring time       |
| [ADR-5](5-time-validation.md)                   | How do we ensure we have valid time         |
| [ADR-6](6-time-storage-in-rule-schedule.md)     | How to handle time in rule schedules        |
| [ADR-7](7-jetstream-store.md)                   | Use NATS Jetstream for the SIOT store       |
