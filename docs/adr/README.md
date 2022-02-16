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

An ADR can have the following sections as needed. The highlighted sections
should probably be in every ADR.

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
| [ADR-3](3-node-lifecycle.md)                    | Node lifecycle                              |
