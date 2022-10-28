# canbus Branch README

The goal is to create a SimpleIoT client that will pull specific data off of a CAN bus and publish it as points, and to create a frontend that will display that data.

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
