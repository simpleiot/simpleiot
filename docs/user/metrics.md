# Metrics

An important part of maintaining healthy systems is to monitor metrics for the
application and system. SIOT can collect metrics for:

- the system
- the SIOT application
- any named processes

For the named process, if there are multiple processes of the same name, then we
add values for all processes found.

## System Metrics

![system-metrics](images/metrics-system.png)

### Thermal Metrics

On Linux systems, the system metrics also include the thermal state of the
board:

- **Temperature** comes from the hwmon sensors and from the thermal zones in
  `/sys/class/thermal`. Both are read because many SoCs expose their board
  sensors through hwmon while reporting the CPU, SoC, and junction temperatures
  through the zones alone. On a Jetson AGX Orin, for example, `tj-thermal` is
  the junction reading that governs throttling.
- **Fan RPM and PWM** come from the hwmon fan and pwm attributes. PWM is the raw
  kernel value, which runs from 0 to 255.
- **Cooling State** is the current state of each entry in
  `/sys/class/thermal/cooling_device*`, keyed by device type. Any value above
  zero means the thermal governor is limiting the system: a `cpufreq` or
  `devfreq` device reports how far the clocks have been pulled back, and a fan
  reports how hard it has been asked to run. Temperature tells you how warm a
  board is, while the cooling state tells you whether that warmth is costing
  performance, so the two are worth reading together. Cooling State Max gives
  the scale each device is measured against and is collected once at startup.

### Power and Clocks

Two more readings round out the picture of how hard a board is working:

- **Voltage, Current, and Power** come from the hwmon power monitors, such as
  the INA3221 devices on a Jetson, and are published in volts, amps, and watts.
  A channel is published when its driver labels it, which is how a board names
  the rail that channel measures, so the points arrive keyed by rail name:
  `VDD_GPU_SOC`, `VIN_SYS_5V0`, and so on. Monitors that do not report power
  themselves still report voltage and current, and the product stands in for the
  missing reading.
- **CPU MHz** is the current clock of each core, keyed by `cpu0`, `cpu1`, and so
  on. Cores that are offline are left out. This is the reading that completes
  the thermal story: temperature says how warm the board is, cooling state says
  the governor stepped in, and the clock says what that cost.

Every reading above is taken on its own, so one that is unavailable, which
happens when a rail is powered down or a monitor channel is disabled, does not
affect the rest. Sensor names are not guaranteed to be unique; repeated names
are numbered, as in `tmp451` and `tmp451_2`.

## SIOT Application Metrics

![app-metrics](images/metrics-app.png)

## Named Process Metrics

![proc-metrics](images/metrics-proc.png)
