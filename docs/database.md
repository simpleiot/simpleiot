+++
title = "Database"
weight = 6
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

We currently use an external InfluxDB 1.x database for storing timeseries data,
but eventually would like to have an embedded timeseries option -- perhaps built
on bolt.
