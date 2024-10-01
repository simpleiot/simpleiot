# File

The file node can be used to store files that are then used by other
nodes/clients. Some examples include the [CAN](can.md) and [Serial](mcu.md)
clients.

The default max payload of NATS is 1MB, so that is currently the file size
limit, but NATS
[can be configured](https://docs.nats.io/reference/faq#is-there-a-message-size-limitation-in-nats)
for a payload size up to 64MB. 8MB is recommended.

See the [Frontend documentation](../ref/frontend.md) for more information how
the file UI is implemented.

If the `Binary` option is selected, the data is base64 encoded before it is
transmitted and stored.
