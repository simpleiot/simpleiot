module github.com/simpleiot/simpleiot

require (
	github.com/adrianmo/go-nmea v1.1.1-0.20190321164421-7572fbeb90aa
	github.com/beevik/ntp v0.3.0
	github.com/benbjohnson/genesis v0.2.1
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/cbrake/influxdbhelper/v2 v2.1.4
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/donovanhide/eventsource v0.0.0-20171031113327-3ed64d21fb0b
	github.com/go-ocf/go-coap v0.0.0-20200224085725-3e22e8f506ea
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.0
	github.com/influxdata/influxdb1-client v0.0.0-20190809212627-fc22c7df067e
	github.com/jacobsa/go-serial v0.0.0-20180131005756-15cf729a72d4
	github.com/nats-io/nats-server/v2 v2.1.8-0.20200601203034-f8d6dd992b71
	github.com/nats-io/nats.go v1.10.1-0.20200606002146-fc6fed82929a
	github.com/timshannon/bolthold v0.0.0-20200316231344-dc30e2b2f90c
	go.bug.st/serial v1.1.0
	go.etcd.io/bbolt v1.3.4
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b
	golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae // indirect
	google.golang.org/protobuf v1.25.0
)

replace github.com/nats-io/nats-server/v2 => github.com/cbrake/nats-server/v2 v2.1.8-0.20200731221538-c267fe885cb5

go 1.13
