# Raspberry Pi GPIO test

A push button and an LED on the Raspberry Pi 40-pin header, used to check the
[GPIO client](../../docs/user/gpio.md) on real hardware. While the button is
held, a rule lights the LED.

## Parts

- Raspberry Pi with a 40-pin header (Pi 3, 4, or 5)
- Normally open push button
- LED
- 330 Ω resistor
- Breadboard and jumper wires

## Wiring

| Signal | GPIO   | Header pin | Connects to                        |
| ------ | ------ | ---------- | ---------------------------------- |
| Switch | GPIO17 | 11         | One side of the push button        |
| Ground | —      | 9          | Other side of the push button      |
| LED    | GPIO27 | 13         | 330 Ω resistor, then the LED anode |
| Ground | —      | 14         | LED cathode (the shorter lead)     |

```
 Pin 11 (GPIO17) ──────┤ button ├────── Pin 9 (GND)

 Pin 13 (GPIO27) ──[ 330 Ω ]──▶|── Pin 14 (GND)
                              LED
```

The switch needs no external resistor. The line uses the internal pull-up, so it
reads high when the button is open and low when it is pressed. `activeLow` turns
that into `value` 1 while the button is held.

GPIO pins are 3.3 V and not 5 V tolerant. Keep the switch between the GPIO and
ground, never 5 V.

## Running

Siot needs access to the `/dev/gpiochip*` devices; run it as root or as a user
in the `gpio` group. Then point it at this directory:

```sh
siot serve -provisioningDir testdata/rpi-gpio
```

Or import the file into an instance that is already running:

```sh
siot import < testdata/rpi-gpio/10-rpi-gpio.yaml
```

The nodes appear under the **RPi GPIO test** group.

## Checking it

1. **Test switch** shows `connected` and a `value` of 0.
2. Press and hold the button. **Test switch** goes to 1 and the LED lights.
3. Release it. The switch goes to 0 and the LED turns off.
4. Writing **Value** on **Test LED** in the UI drives the LED directly.

## Selecting the chip

The file selects the header's GPIO chip by its label, `pinctrl-rp1`, which is
the Pi 5. The chip number varies with the kernel and device tree (one Pi 5 build
puts the header on `gpiochip15`), and the label does not. On other models,
change `chip` in the file:

| Model | `chip`            |
| ----- | ----------------- |
| Pi 5  | `pinctrl-rp1`     |
| Pi 4  | `pinctrl-bcm2711` |
| Pi 3  | `pinctrl-bcm2835` |

If a node reports an error, `gpiodetect` lists the chips and their labels, and
`gpioinfo` shows whether `GPIO17` and `GPIO27` are free.
