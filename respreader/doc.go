/*
Package respreader provides a convenient way to frame read data from devices
that use
prompt/response protocols such as Modbus (and other RS485 protocols) and modem
AT commands. The fundamental assumption is a device takes some variable amount of
time to respond to a request, formats up a packet, and then streams it out the
serial port. Once the response data starts streaming, any significant gap with
no data indicates the response is complete. Once this gap is detected, a Read()
returns with accumulated data.

This method of framing a response has the following advantages:

1) minimizes the wasted timing waiting for a response to the chunkTimeout defined
below. More simplistic implementations may take the worste case response time
for all packets and simply wait that amount of time for the response to come.
This works, but the bus is tied up during the wait that could be used for
more packets.

2) It is simple in that you don't have to parse the response on the fly to determine
when it is complete.

The obvious disadvantage of this method of framing is that the device may insert a
significant delay in sending the response that will cause the reader to think the
resonse is complete. As long as this delay is still significantly shorter than
the overall response time, it still can work fairly well. Some experiementation may
be required to optimize the chunkTimeout setting.

Example using a serial port:

	import (
		"io"

		"github.com/jacobsa/go-serial/serial"
		"github.com/simpleiot/simpleiot/respreader"
	)

	options := serial.OpenOptions{
		PortName:              "/dev/ttyUSB0",
		BaudRate:              9600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 0,
		RTSCTSFlowControl:     true,
	}

	port, err := serial.Open(options)

	port = respreader.NewResponseReadWriteCloser(port, time.Second,
	time.Millisecond * 50)

	// send out prompt
	port.Write("ATAI")

	// read response
	data := make([]byte, 128)
	count, err := port.Read(data)
	data = data[0:count]

	// now process response ...

*/
package respreader
