/*
Package client contains utilties for creating Simple IoT clients.

A Simple IoT Client contains the logic for nodes described in the Simple IoT graph store.
Examples of built in clients include: Modbus, 1-wire, Upstream, etc.

A client [Manager] is used to watch the Simple IoT store and create and update new clients
as they are created and modified.

See [Client] and [NewManager] for ideas on how to get started with your own client.

When debugging client test code, it can be very useful to dump the node tree for inspection.
This can be done with the following code:

	nodes, err := client.ExportNodes(nc, "root")
	if err != nil {
		t.Fatal("Error exporting nodes: ", err)
	}

	fmt.Println(string(nodes))

This package also contains a number of utility functions for interacting with the
Simple IoT [NATS API].

[NATS API]: https://docs.simpleiot.org/docs/ref/api.html#nats
*/
package client
