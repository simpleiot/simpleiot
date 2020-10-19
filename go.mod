module github.com/simpleiot/simpleiot

require (
	github.com/adrianmo/go-nmea v1.1.1-0.20190321164421-7572fbeb90aa
	github.com/beevik/ntp v0.3.0
	github.com/benbjohnson/genesis v0.2.1
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/cbrake/influxdbhelper/v2 v2.1.4
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/donovanhide/eventsource v0.0.0-20171031113327-3ed64d21fb0b
	github.com/genjidb/genji v0.8.0
	github.com/genjidb/genji/cmd/genji v0.8.0 // indirect
	github.com/go-ocf/go-coap v0.0.0-20200224085725-3e22e8f506ea
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.0
	github.com/inconshreveable/log15 v0.0.0-20200109203555-b30bc20e4fd1 // indirect
	github.com/influxdata/influxdb1-client v0.0.0-20190809212627-fc22c7df067e
	github.com/jacobsa/go-serial v0.0.0-20180131005756-15cf729a72d4
	github.com/kevinburke/go-types v0.0.0-20200309064045-f2d4aea18a7a // indirect
	github.com/kevinburke/go.uuid v1.2.0 // indirect
	github.com/kevinburke/rest v0.0.0-20200429221318-0d2892b400f8 // indirect
	github.com/kevinburke/twilio-go v0.0.0-20200810163702-320748330fac
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/nats-io/nats-server/v2 v2.1.8-0.20200814173904-d30550166e2f
	github.com/nats-io/nats.go v1.10.1-0.20200720131241-97eff70ce747
	github.com/timshannon/bolthold v0.0.0-20200316231344-dc30e2b2f90c
	github.com/ttacon/builder v0.0.0-20170518171403-c099f663e1c2 // indirect
	github.com/ttacon/libphonenumber v1.1.0 // indirect
	go.bug.st/serial v1.1.0
	go.etcd.io/bbolt v1.3.5
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b
	google.golang.org/protobuf v1.25.0
)

replace github.com/nats-io/nats.go => github.com/cbrake/nats.go v1.10.1-0.20200817210920-7a8e05e18c84

go 1.13
