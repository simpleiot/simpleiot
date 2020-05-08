# Simple IoT Development

This document attempts to outlines the basic architecture and development
philosophy. The basics are covered in the [readme](readme.md). As the name
suggests, a core value of the project is simplicity. Thus any changes should be
made with this in mind. This project is far from perfect and there are likely
many better ways to do things.

## Frontend architecture

Much of the frontend architecture is already defined by the Elm architecture.
However, we still have to decide how data flows between various modules in the
frontend. If possible, we'd like to keep the UI
[optimistic](https://blog.meteor.com/optimistic-ui-with-meteor-67b5a78c3fcf) if
possible. Thoughts on how to accomplish this:

- single data model at top level
- modifications to the backend database are sent to the top level, the model is
  modified first, and then a request is sent to the backend to modify the
  database. This ensures the value does not flash or revert to old value while
  the backend request is being made.

## Backend architecture
