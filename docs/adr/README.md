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

Discussion should happen in the PR. The main reason for this is we can comment
inline with the code or ADR document and helps keep the conversation centered
around the code/documentation.

## ADRs

| Index                                           | Description                                 |
| ----------------------------------------------- | ------------------------------------------- |
| [ADR-1](1-consider-changing-point-data-type.md) | Consider changing/expanding point data type |
