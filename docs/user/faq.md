# Frequently Asked Questions

### Q: How is SIOT different than Home Assistant, OpenHAB, Domoticz, etc.?

Although there may be some overlap and Simple IoT may eventually support a
number of off the shelf consumer IoT devices, the genesis, and intent of the
project is for developing IoT products and the infrastructure required to
support them.

### Q: How is SIOT different than Particle.io, etc.?

Particle.io provides excellent infrastructure to support their devices and solve
many of the hard problems such as remote firmware update, getting data securely
from device to cloud, and efficient data bandwidth usage. But, they don't
provide a way to provide a user facing portal for a product that customers can
use to see data and interact with the device.

### Q: How is SIOT different than AWS/Azure/GCP/... IoT?

SIOT is designed to be simple to develop and deploy without a lot of moving
parts. We've reduced an IoT system to a
[few basic concepts](https://github.com/simpleiot/simpleiot/tree/master#core-ideas)
that are exactly the same in the cloud and on edge devices. This symmetry is
powerful and allows us to easily implement and move functionality wherever it is
needed. If you need
[Google Scale](https://blog.bradfieldcs.com/you-are-not-google-84912cf44afb),
SIOT may not be the right choice; however, for smaller systems where you want a
system that is easier to develop, deploy, and maintain, consider SIOT.

### Q: Can't NATS JetStream do everything SIOT does?

This is a good question and I'm not sure yet. NATS has some very interesting
features like JetStream which can queue data and store data in a key-value store
and data can be synchronized between instances. NATS also has a concept of
leaf-nodes, which conceptually makes sense for edge/gateway connections.
JetStream is optimized for data flowing in one direction (ex: orders through
fulfillment). SIOT is optimized for data flowing in any direction and data is
merged using data structures with CRDT (conflict-free replicated data types)
properties. SIOT also stores data in a DAG (directed acyclic graph) which allows
a node to be a child of multiple nodes, which is difficult to do in a
hierarchical namespace. Additionally, each node is defined by an array of points
and modifications to the system are communicated by transferring points. SIOT is
a batteries included complete solution for IoT solutions, including a web
framework, clients for various types of IO (ex: Modbus) and cloud services (ex:
Twilio). We will continue to explore using more of NATS core functionality as we
move forward.
