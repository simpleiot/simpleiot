# MCU Devices

Status: Specification

Microcontroller (MCU) devices can be connected to Simple IoT systems via various
serial transports (RS232, RS485, CAN, and USB Serial). The
[Arduino](https://www.arduino.cc/) platform is one example of a MCU platform
that is easy to use and program. Simple IoT provides a serial interface module
that can be used to interface with these systems. The combination of a laptop or
a Raspberry PI makes a useful lab device for monitoring analog and digital
signals. Data can be logged to InfluxDB and viewed in the InfluxDB Web UI or
Grafana. This concept can be scaled into products where you might have a Linux
MPU handling data/connectivity and a MCU doing real-time control.

![mcu](images/mcu.png)

TODO: add instructions for setting up an Arduino system.

## Debug Levels

You can set the following debug levels to log information.

- 0: no debug information
- 1: log ASCII strings (must be COBS wrapped)
- 2: log raw data (must be COBS wrapped)
