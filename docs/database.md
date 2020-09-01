---
id: database
title: Database
sidebar_label: Database
---

Currently, Simple IoT supports [bbolt](https://github.com/etcd-io/bbolt) as a
data store. This is an embedded keyvalue store that is used similar to how NoSQL
databases are used. [Bolthold](https://github.com/timshannon/bolthold) is used
to provide some convenience for storing and querying Go types on top of Bolt.

The current database schema is several MongoDb schema design posts
([1](https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design-part-1),
[2](https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design-part-2),
[3](https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design-part-3)).

Collections in the database relate to each other as shown below. Only fields
that contain relational data are shown.

- Users
- Groups
  - Users []{UserID, []Role}
- Devices
  - Points
    - Devices []DeviceID
  - Groups []GroupID
  - Parents []DeviceID

The idea is a Group will have a limitted number of users, so it is OK to embed
IDs. However, a group may contain 1000's of devices, but a Device will belong to
a limitted number of groups, so it is best to embed the Group ID in the device.

This is only for the current Bolt datastore -- after we create an interface for
the data store, this could be optimized in any way necessary.
