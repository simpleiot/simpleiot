# Client Development

Most functionality in Simple IoT is implemented in Clients.

![app-arch](images/arch-app.png)

Each client is configured by one or more nodes in the SIOT store graph. These
nodes may be created by a user, a process that detects new plug and play
hardware, or other clients.

A client interacts with the system by listening for new points it is interested
in and sending out points as it aquires new data.

Simple IoT provides a utilites that assist in creating new clients. See the
[Go package documentation](https://pkg.go.dev/github.com/simpleiot/simpleiot/client)
for more information. A client manager is created for each client type. This
manager instantiates new client instances when new nodes are detected and then
sends point updates to the client.
