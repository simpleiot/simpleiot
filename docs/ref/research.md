# Research

This document contains information that has been researched during the course of
creating Simple IoT.

> **Status:** this page is a historical record of the exploration that shaped
> the design; it is not a description of how Simple IoT works today. The
> hash-tree synchronization discussed below was implemented and later replaced
> by replicating JetStream streams, which carry their own sequence numbers and
> avoid the moving-target comparison problem entirely. See
> [ADR-7](../adr/7-jetstream-store.md) for the analysis and decision, and the
> [store](store.md) and [synchronization](sync.md) references for the current
> design.

## Synchronization

An IoT system is inherently distributed. At a minimum, there are three
components:

1. Device (Go, C, etc.)
1. Cloud (Go)
1. Multiple browsers (Elm, JavaScript)

Data can be changed in any of the above locations and must be seamlessly
synchronized to other locations. Failing to consider this simple requirement
early in building the system can make for brittle and overly complex systems.

### The moving target problem

As long as the connection between instances is solid, they will stay
synchronized as each instance will receive all points it is interested in.
Therefore, verifying synchronization by comparing Node hashes is a backup
mechanism - that allows us to see what changed when disconnected. The root
hashes for a downstream instance changes every time anything in that system
changes. Only one value needs to be compared to ensure your entire config is
synchronized, but it is also a disadvantage in that the top level hash is
changing more often so you are trying to compare two moving targets. This is not
a problem if things are changing slow enough that it does not matter if they are
changing. However, this also limits the data rates to which we can scale.

Some systems use a concept called Merkle clocks, where events are stored in a
Merle DAG and existing nodes in the DAG are immutable and new events are always
added as parents to existing events. An immutable DAG has an advantage in that
you can always work back in history, which never changes. The SIOT Node tree is
mutable by definition. Actual budget uses a similar concept in that it uses a
Merkle Trie to represent events in time and then prunes the tree as time goes
on.

We could create a separate structure to sync all events (points), but that would
require a separate structure on the server for every downstream device and seems
overly complex.

Is it critical that we see all historical data? In an IoT system, there are
essentially two sets of date - current state/config, and historical data. The
current state is most critical for most things, but historical data may be used
for some algorithms and viewed by users. The volume of data makes it impractical
to store all data in resource constrained edge systems. However, maybe it's a
mistake to separate these two as synchronizing all data might simplify the
system.

One way to handle the moving target problem is to store an array of previous
hashes for the device node in both instances - perhaps for as long as the
synchronization interval. The downstream could then fetch the upstream hash
array and see if any of the entries match an entry in the downstream array. This
would help cover the case where there may be some time difference when things
get updated, but the history should be similar. If there is a hash in history
that matches, then we are probably OK.

Another approach would be to track metrics on how often the top level hash is
updating - if it is too often, then perhaps the system needs tuned.

There could also be some type of stop-the-world lock where both systems stop
processing new nodes during the sync operation. However, if they are not in
sync, this probably won't help and definitely hurts scalability.

### Resgate

[resgate.io](https://resgate.io) is an interesting project that solves the
problem of creating a real-time API gateway where web clients are synchronized
seamlessly. This project uses NATS.io for a backbone, which makes it interesting
as NATS is core to this project.

The Resgate system is primarily concerned with synchronizing browser contents.

### Couch/pouchdb

Has some interesting ideas.

### Merkle Trees

- https://en.wikipedia.org/wiki/Merkle_tree
- https://jack-vanlightly.com/blog/2016/10/24/exploring-the-use-of-hash-trees-for-data-synchronization-part-1
- https://www.codementor.io/blog/merkle-trees-5h9arzd3n8
  - Version Control Systems Version control systems like Git and Mercurial use
    specialized Merkle trees to manage versions of files and even directories.
    One advantage of using Merkle trees in version control systems is we can
    simply compare hashes of files and directories between two commits to know
    if they've been modified or not, which is quite fast.
  - No-SQL distributed database systems like Apache Cassandra and Amazon
    DynamoDB use Merkle trees to detect inconsistencies between data replicas.
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
- NATS
  [key/value store](https://docs.nats.io/using-nats/developer/develop_jetstream/kv)

### Distributed Hash Tables

- https://en.wikipedia.org/wiki/Distributed_hash_table

### CRDT (Conflict-free replicated datatype)

- https://en.wikipedia.org/wiki/Conflict-free_replicated_data_type
- [Yjs](https://yjs.dev/#community)
  - https://blog.kevinjahns.de/are-crdts-suitable-for-shared-editing/
- https://tantaman.com/2022-10-18-lamport-sufficient-for-lww.html

### Databases

- https://tantaman.com/2022-08-23-why-sqlite-why-now.html
  - instead of doing:
    `select comment.* from post join comment on comment.post_id = post.id where post.id = x and comment.date < cursor.date and comment.id < cursor.id order by date, id desc limit 101`
  - we do: `post.comments().last(10).after(curosr);`

### Timestamps

- [Lamport timestamp](https://en.wikipedia.org/wiki/Lamport_timestamp)
  - used by Yjs

## Other IoT Systems

### AWS IoT

- https://www.thingrex.com/aws_iot_thing_attributes_intro/
  - Thing properties include the following, which are analogous to SIOT node
    fields.
    - Name (Description)
    - Type (Type)
    - Attributes (Points)
    - Groups (Described by tree structure)
    - Billing Group (Can also be described by tree structure)
- https://www.thingrex.com/aws_iot_thing_type/
  - Each type has a specified attributes - kind of a neat idea

## Industry 4.0

### umati

[umati](https://umati.org/) (Universal Machine Technology Interface) is a
community of machine builders, component suppliers, software vendors, and users
working toward open interfaces for machine data. It builds on
[OPC UA](https://opcfoundation.org/about/opc-technologies/opc-ua/) rather than
defining a protocol of its own.

- the data models are published as OPC UA companion specifications -- around 25
  of them so far, covering machine tools, robotics, measurement systems, and
  plastics machinery, with roughly 30 more in progress
- the central "OPC UA for Machinery" specification carries the building blocks
  common to any machine: identification, job management, and energy monitoring
- https://umati.app/ is a live demonstration platform showing data from
  participating machines
- interesting to Simple IoT as a ready-made vocabulary for naming points on
  factory equipment, and as further reason to build a native OPC UA client; see
  the [OPC UA notes](../user/plc.md#opc-ua-planned)
