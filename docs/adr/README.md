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
the Reference Guide documentation.

An ADR can have the following sections as needed. The highlighted sections
should probably be in every ADR.

- Header
  - Author: AUTHOR_NAME Last updated: 2021-11-04
  - Issue:
  - PR/Discussion:
  - Status: \[Discussion|Proposed|Accepted|Deprecated|Superseded\]
- Problem -- what problem are we trying to solve?
- **Context** -- background, facts surrounding this discussion.
- Design -- discussion on implementation -- may present several different
  options.
- **Decision** -- what was decided.
  - Objections/concerns
- **Consequences** -- what is the impact, both negative and positive.
- Additional Notes/Reference -- links to reference material that may be
  relevant.

## ADRs

| Index                                           | Description                                 |
| ----------------------------------------------- | ------------------------------------------- |
| [ADR-1](1-consider-changing-point-data-type.md) | Consider changing/expanding point data type |
| [ADR-2](2-authz.md)                             | Authorization considerations.               |
| [ADR-3](3-node-lifecycle.md)                    | Node lifecycle                              |
