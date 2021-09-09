+++
title = "Research"
weight = 100
+++

This document contains information that has been researched during the course of
creating Simple IoT.

## Synchronization

An IoT system is inherently distributed. At a minimum, there are three
components:

1. device (Go, C, etc.)
1. cloud (Go)
1. multiple browsers (Elm, Js)

Data can be changed in any of the above locations and must be seamlessly
synchronized to other locations. Failing to consider this simple requirement
early in building the system can make for brittle and overly complex systems.

### Resgate

[resgate.io](https://resgate.io) is an interesting project that solves the
problem of creating a real-time API gateway where web clients are synchronized
seamlessly. This project uses NATS.io for a backbone, which makes it interesting
as NATS is core to this project.

The Resgate system is primarily concerned with synchronizing browser contents.

### Couch/pouchdb

Has some interesting ideas.

### Merkle Trees

- https://medium.com/@rkkautsar/synchronizing-your-hierarchical-data-with-merkle-tree-dbfe37db3ab7
- https://en.wikipedia.org/wiki/Merkle_tree
- https://jack-vanlightly.com/blog/2016/10/24/exploring-the-use-of-hash-trees-for-data-synchronization-part-1
- https://www.codementor.io/blog/merkle-trees-5h9arzd3n8
  - Version Control Systems Version control systems like Git and Mercurial use
    specialized merkle trees to manage versions of files and even directories.
    One advantage of using merkle trees in version control systems is we can
    simply compare hashes of files and directories between two commits to know
    if they've been modified or not, which is quite fast. This
    [post](https://blog.sourced.tech/post/difftree/) discusses in detail how the
    entire process works.
  - No-SQL distributed database systems like Apache Cassandra and Amazon
    DynamoDB use merkle trees to detect inconsistencies between data replicas.
    This process of repairing the data by comparing all replicas and updating
    each one of them to the newest version is also called anti-entropy repair.
    The process is also described in
    [Cassandra's documentation](https://docs.datastax.com/en/cassandra/3.0/cassandra/operations/opsRepairNodesManualRepair.html).

#### Scaling Merkel trees

One limitation of Merkel trees is the difficulty of updating the tree
concurrently. Some information on this:

- [how to scale blockchains](https://www.forbes.com/sites/forbestechcouncil/2018/11/27/sidechains-how-to-scale-and-improve-blockchains-safely/?sh=193537e64418)
- [Angela: A Sparse, Distributed, and Highly Concurrent Merkle Tree](https://people.eecs.berkeley.edu/~kubitron/courses/cs262a-F18/projects/reports/project1_report_ver3.pdf)

### Distributed key/value databases

- etcd

### Distributed Hash Tables

- https://en.wikipedia.org/wiki/Distributed_hash_table

### CRDT (Conflict-free replicated data type)

- https://en.wikipedia.org/wiki/Conflict-free_replicated_data_type
- [Yjs](https://yjs.dev/#community)
  - https://blog.kevinjahns.de/are-crdts-suitable-for-shared-editing/

### Timestamps

- [Lamport timestamp](https://en.wikipedia.org/wiki/Lamport_timestamp)
  - used by Yjs
