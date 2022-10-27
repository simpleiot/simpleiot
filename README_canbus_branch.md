# canbus Branch README

The goal is to create a SimpleIoT client that will pull specific data off of a CAN bus and publish it as points, and to create a frontend that will display that data.

Steps:
- [ ] copy the Go serial client and modify to work with CAN
      - [ ] add serial to the list of default clients in clients.go and build/test
      - [ ] allow configuration of bus name, baud rate, etc. instead of Serial parameters
      - [ ] create can.go in clients/ to hold the can client
      - [ ] add can to the list of default clients in clients.go
- [ ] copy the Elm Serial Node and modify to display data in a simple way
      - [ ] allow configuration of bus name, baud rate, etc. instead of Serial parameters
- [ ] modify CAN client to recieve data in protobuf format through J1939 multi-packet messages (nanopb)
