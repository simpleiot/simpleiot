# Simple IoT Store

We currently use SQLite to implement the persistent store for Simple IoT. Each
instance (cloud, edge, etc.) has its own store that must be synchronized with
replicas of the data located in other instances.

## Reasons for using SQLite

We have evaluated BoltDB, Genji, and various other Go key/value stores in the
past and settled on SQLite for the following reasons:

- **Reliability**: SQLite is very well tested and
  [handles things](https://www.sqlite.org/transactional.html) like program/OS
  crashes, power failures, etc. It is important that the configuration for a
  system never become corrupt to the point where it won't load.
- **Stable file format**: Dealing with file format changes is not something we
  want to deal with when we have 100's of systems in the field. A SQLite file is
  very portable across time and between systems.
- **Pure Go**: There is now a
  [pure Go version](https://pkg.go.dev/modernc.org/sqlite) of SQLite. If more
  performance is needed or smaller binary size, the native version of SQLite can
  still be used.
- **The relational model**: it seems to make sense to store points and nodes in
  separate tables. This allows us to update points more quickly as it is a
  separate line in the DB. It also seems like flat data structures are generally
  a good thing versus deeply nested objects.
- **Fast**: SQLite does read caching, and other things that make it quite fast.
- **Lots of innovation around SQLite**:
  [LiteFS](https://github.com/superfly/litefs),
  [Litestream](https://litestream.io/), etc.
- **Multi-process**: SQLite
  [supports multiple processes](https://www.sqlite.org/faq.html#q5). While we
  don't really need this for core functionality, it is very handy for debugging,
  and there may be instances where you need multiple applications in your stack.
