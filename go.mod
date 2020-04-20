module github.com/simpleiot/simpleiot

require (
	github.com/adrianmo/go-nmea v1.1.1-0.20190321164421-7572fbeb90aa
	github.com/benbjohnson/genesis v0.2.1
	github.com/cbrake/influxdbhelper/v2 v2.1.4
	github.com/creack/goselect v0.1.1 // indirect
	github.com/donovanhide/eventsource v0.0.0-20171031113327-3ed64d21fb0b
	github.com/go-ocf/go-coap v0.0.0-20200406073902-cf923db524db
	github.com/goburrow/modbus v0.1.0
	github.com/goburrow/serial v0.1.0
	github.com/gorilla/websocket v1.4.0
	github.com/influxdata/influxdb1-client v0.0.0-20190809212627-fc22c7df067e
	github.com/jacobsa/go-serial v0.0.0-20180131005756-15cf729a72d4
	github.com/tbrandon/mbserver v0.0.0-20170611213546-993e1772cc62
	github.com/timshannon/bolthold v0.0.0-20180829183128-83840edea944
	go.bug.st/serial v1.0.0
	go.bug.st/serial.v1 v0.0.0-20191202182710-24a6610f0541 // indirect
	go.etcd.io/bbolt v1.3.4 // indirect
)

go 1.13

replace github.com/tbrandon/mbserver => ../mbserver

replace github.com/goburrow/modbus => ../modbus

replace go.bug.st/serial => ../go-serial
