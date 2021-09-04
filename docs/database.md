+++
title = "Database"
weight = 7
+++

Currently, Simple IoT supports [bbolt](https://github.com/etcd-io/bbolt) as a
data store. This is an embedded keyvalue store that is used similar to how NoSQL
databases are used. [Genji](https://genji.dev/) is used to provide some
convenience for storing and querying Go types on top of Bolt.

The current database schema is several MongoDb schema design posts
([1](https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design-part-1),
[2](https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design-part-2),
[3](https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design-part-3)).

As described in the [architecture](architecture.md) document, nodes and edges
are the primary data structures stored in database.

We currently use an external InfluxDB 2.x database for storing timeseries data,
but eventually would like to have an embedded timeseries option -- perhaps built
on bolt.

## Evolvability

One important consideration in data design is the can the system be easily
changed. With a distributed system, you may have different versions of the
software running at the same time using the same data. One version may use/store
additional information that the other does not. In this case, it is very
important that the other version does not delete this data, as could easily
happen if you decode data into a type, and then re-encode and store it.

With the Node/Point system, we don't have to worry about this issue because
Nodes are only updated by sending Points. It is not possible to delete a Node
Point. So it one version writes a Point the other is not using, it will be
transferred, stored, syncronized, etc and simply ignored by version that don't
use this point. This is another case where SIOT solves a hard problem that
typically requires quite a bit of care and effort.
