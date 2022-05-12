# 1-Wire

1-Wire is a device communication bus that provides low-speed data over a single
conductor. It is also possible to power some devices over the data signal as
well, but often a third wire is run for power.

## Bus Controllers

There are a number of bus controllers available but one of the simplest is a
GPIO on a Raspberry PI. To enable, add the following to the `/boot/config.txt`
file:

`dtoverlay=w1-gpio`

This enables a 1-wire bus on GPIO 4.

To add a bus to a different pin:

`dtoverlay=w1-gpio,gpiopin=x`

A 4.6kΩ pull-up resistor is needed between the 1-wire signal and 3.3V.

See [this page](https://pinout.xyz/pinout/1_wire#) for more information.

TODO: add schematic

## Supported devices

Simple IoT currently supports 1-wire temperature sensors such as the DS18B20.
This is a very popular and practical digital temperature sensor. Each sensor has
a unique address so you can address a number of them using a single 1-wire port.
