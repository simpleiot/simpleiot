# MCU Devices

Microcontroller (MCU) devices can be connected to Simple IoT systems via various
serial transports (RS232, RS485, CAN, and USB Serial). The
[Arduino](https://www.arduino.cc/) platform is one example of a MCU platform
that is easy to use and program. Simple IoT provides a serial interface module
that can be used to interface with these systems. The combination of a laptop or
a Raspberry PI makes a useful lab device for monitoring analog and digital
signals. Data can be logged to InfluxDB and viewed in the InfluxDB Web UI or
Grafana. This concept can be scaled into products where you might have a Linux
MPU handling data/connectivity and a MCU doing real-time control.

See the [Serial reference documentation](../ref/serial.md) for more technical
details on this client.

![mcu](images/mcu.png)

## Arduino Examples

Several
[Arduino examples](https://github.com/simpleiot/firmware/tree/master/Arduino)
are available that can be used to demonstrate this functionality.

See [reference documentation](../ref/serial.md) for more information.

TODO: add instructions for setting up an Arduino system.

## Debug Levels

You can set the following debug levels to log information.

- 0: no debug information
- 1: log ASCII strings (must be COBS wrapped) (typically used for debugging code
  on the MCU)
- 4: log points received or sent to the MCU
- 8: log cobs decoded data (must be COBS wrapped)
- 9: log raw serial data received (pre-COBS)
