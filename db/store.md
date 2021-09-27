# Simple IoT Store

## Locking

Currently, the store is made of up:

- data stored to disk in a database (Genji)
- a node cache
- an edge cache

The caches are used to speed up read to nodes as loading them from the database
is an expensive operation. We need to manage locking for concurrent access to
the cache. Managing locking can be tricky as you need to figure out at what
level to do the locking:

1. lock entire functions at a high level
1. only lock bits of code that access the cache data structures

It seems locking bits of code that access the data structures is more
maintainable as then you can nest functions without worrying about deadlocks
where the entire function locks.
