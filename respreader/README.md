# Response Reader

Package respreader provides a convenient way to read data from devices that use
prompt/response protocols such as Modbus (and other RS485 protocols) and modem
AT commands. The fundamental assumption is a device takes some variable amount of
time to respond to a request, formats up a packet, and then streams it out the
serial port. Once the response data starts streaming, and significant gap with
no data indicates the response is complete.

see https://godoc.org/github.com/simpleiot/simpleiot/respreader for
documentation
