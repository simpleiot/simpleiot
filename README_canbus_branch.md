# canbus Branch README

The goal is to create a SimpleIoT client that will pull specific data off of a CAN bus and publish it as points, and to create a frontend that will display that data.

## Example Usage

### Test Program:

Copy this code to a Go file on your Linux machine in a folder by itself, and run `go get`
to pull the dependencies.


```go
package main

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// exNode is decoded data from the client node
type exNode struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Port        int    `point:"port"`
	Role        string `edgepoint:"role"`
}

// exNodeClient contains the logic for this client
type exNodeClient struct {
	nc            *nats.Conn
	config        client.SerialDev
	stop          chan struct{}
	stopped       chan struct{}
	newPoints     chan client.NewPoints
	newEdgePoints chan client.NewPoints
	chGetConfig   chan chan client.SerialDev
}

// newExNodeClient is passed to the NewManager() function call -- when
// a new node is detected, the Manager will call this function to construct
// a new client.
func newExNodeClient(nc *nats.Conn, config client.SerialDev) client.Client {
	return &exNodeClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan client.NewPoints),
		newEdgePoints: make(chan client.NewPoints),
	}
}

// Start runs the main logic for this client and blocks until stopped
func (tnc *exNodeClient) Start() error {
	for {
		select {
		case <-tnc.stop:
			close(tnc.stopped)
			return nil
		case pts := <-tnc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &tnc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
			log.Printf("New config: %+v\n", tnc.config)
		case pts := <-tnc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &tnc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}
		case ch := <-tnc.chGetConfig:
			ch <- tnc.config
		}
	}
}

// Stop sends a signal to the Start function to exit
func (tnc *exNodeClient) Stop(err error) {
	close(tnc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (tnc *exNodeClient) Points(id string, points []data.Point) {
	tnc.newPoints <- client.NewPoints{id, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (tnc *exNodeClient) EdgePoints(id, parent string, points []data.Point) {
	tnc.newEdgePoints <- client.NewPoints{id, parent, points}
}

func main() {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		log.Println("Error starting test server: ", err)
	}

	defer stop()

	canBusTest := client.CanBus{
		ID:          "ID-canBus",
		Parent:      root.ID,
		Description: "vcan0",
		Interface:   "vcan0",
		DbFilePath:  "test.kcd",
	}

	err = client.SendNodeType(nc, canBusTest, "test")
	if err != nil {
		log.Println("Error sending CAN node: ", err)
	}

	// Create a new manager for nodes of type "testNode". The manager looks for new nodes under the
	// root and if it finds any, it instantiates a new client, and sends point updates to it
	m := client.NewManager(nc, newExNodeClient)
	m.Start()

	// Now any updates to the node will trigger Points/EdgePoints callbacks in the above client
}
```

### Setup Virtual CAN Interface

Run this in the command line. [Reference](https://www.pragmaticlinux.com/2021/10/how-to-create-a-virtual-can-interface-on-linux/)

```bash
sudo modprobe vcan0
sudo ip link add dev vcan0 type vcan
sudo ip link set up vcan0
```

### Create the CAN Database

Create a file in the folder with the Go code named "test.kcd" containing the following:
```xml
<NetworkDefinition xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://kayak.2codeornot2code.org/1.0" xsi:schemaLocation="Definition.xsd">
  <Document name="Some Document Name">some text</Document>
  <Bus name="sampledatabase">
    <Message id="0x123" name="HelloWorld" length="8">
      <Notes></Notes>
      <Signal name="Hello" offset="0" length="8"/>
      <Signal name="World" offset="8" length="8"/>
    </Message>
    <Message id="0x12345678" name="Food" length="8" format="extended">
      <Notes></Notes>
      <Signal name="State" offset="0" length="32"/>
      <Signal name="Type" offset="32" length="32"/>
    </Message>
  </Bus>
</NetworkDefinition>
```

You can create any CAN database you want by crafting it in Kvaser's free DBC editor and
then using the `canmatrix` tool to convert it to KCD format. Note that `canmatrix` does
not support all features of the DBC and KCD formats.

### Run it!

`go run <file.go>`

In a separate terminal:
```
cansend vcan0 123#R{8}
cansend vcan0 12345678#DEADBEEF
```

You should see a log like this:
```
2022/12/05 15:03:23 NATS server, port: 8900, http port: 8902, auth enabled: no
2022/12/05 15:03:23 NATS server WS enabled on port: 8903
2022/12/05 15:03:23 Open store:  test.sqlite?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(8000)&_pragma=journal_size_limit(100000000)
2022/12/05 15:03:23 STORE: Initialize root node and admin user
2022/12/05 15:03:23 store connecting to nats server:  nats://localhost:8900
2022/12/05 15:03:23 Starting http server, debug:  false
2022/12/05 15:03:23 Starting portal on port:  8901
2022/12/05 15:03:24 Nats server client reconnect
2022/12/05 15:03:24 Setting app version:  
2022/12/05 15:03:24 Error reading OS version:  searching /etc/os-release, got: Invalid character(s) found in major number "0Ubuntu"
2022/12/05 15:03:24 CanBusClient: Starting CAN bus client:  vcan0
2022/12/05 15:03:24 CanBusClient: read msg 123 sig Hello: start=0 len=8 scale=0 offset=0 unit=
2022/12/05 15:03:24 CanBusClient: read msg 123 sig World: start=8 len=8 scale=0 offset=0 unit=
2022/12/05 15:03:24 CanBusClient: read msg 12345678 sig State: start=0 len=32 scale=0 offset=0 unit=
2022/12/05 15:03:24 CanBusClient: read msg 12345678 sig Type: start=32 len=32 scale=0 offset=0 unit=
2022/12/05 15:04:11 CanBusClient: got 123#R
2022/12/05 15:04:11 CanBusClient: created point HelloWorld.Hello() 0
2022/12/05 15:04:11 CanBusClient: created point HelloWorld.World() 0
2022/12/05 15:04:11 CanBusClient: successfully sent points
2022/12/05 15:04:50 CanBusClient: got 12345678#DEADBEEF
2022/12/05 15:04:50 CanBusClient: created point Food.State() 4.022250974e+09
2022/12/05 15:04:50 CanBusClient: created point Food.Type() 0
2022/12/05 15:04:50 CanBusClient: successfully sent points
```



## Documentation

The `can` client works as follows:
- One or many MCU's send CAN packages to an MPU running the `can` client
- The `can` client is configured to accept certain PGN's and translate each SPN contained in them to a SimpleIoT Point that is sent out through NATS
  - The `can` client will recieve multiple messages per processor cycle
    - If they are read in by a go routine and then passed over a channel to the select loop, how do we read and pass multiple messages per processor cycle
    - How large should the channel buffer be? We don't want to get behind and process old information in general.
- The frontend looks for and displays the points recieved

In the future the `can` client will expect incoming CAN data to be in protobuf format:
- One or many MCU's send a specific J1939 multi-packet CAN message to an MPU running the `can` client. The data in this multi-packet message (only one PGN, source address may vary) is structured in protobuf format containing SimpleIoT Points
  - At least one of MCU's must be able to send out protobuf data as a J1939 multi-packet CAN message.
  - This MCU must translate data from other MCU's that are not capable of this and rebroadcast it
  - The CAN bus utilization of the existing bus must be low enough to support this rebroadcasting or the translating MCU must have a separate bus available
- The Points are sent out directly through NATS by the `can` client, the `can` client is fully generic and doesn't care what type of data it recieves over CAN as long as it contains Points.
- The frontend looks for and displays the points recieved

## Steps:

copy the Go serial client and modify to work with CAN
- [ ] add serial to the list of default clients in clients.go and build/test
- [ ] allow configuration of bus name, baud rate, etc. instead of Serial parameters
- [ ] create can.go in clients/ to hold the can client
- [ ] add can to the list of default clients in clients.go

copy the Elm Serial Node and modify to display data in a simple way
- [ ] allow configuration of bus name, baud rate, etc. instead of Serial parameters

modify CAN client to recieve data in protobuf format through J1939 multi-packet messages (nanopb)

## Steps to add a CAN bus node
- Top.elm
  - set shouldDisplay -> true
- Node.elm
- Point.elm
- data/schema.go
- client/can.go
