module github.com/simpleiot/simpleiot

require (
	github.com/DataDog/zstd v1.4.8 // indirect
	github.com/adrianmo/go-nmea v1.1.1-0.20190321164421-7572fbeb90aa
	github.com/beevik/ntp v0.3.0
	github.com/benbjohnson/genesis v0.2.1
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/dgraph-io/badger/v3 v3.2103.0
	github.com/dgraph-io/ristretto v0.1.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/donovanhide/eventsource v0.0.0-20171031113327-3ed64d21fb0b
	github.com/genjidb/genji v0.12.0
	github.com/genjidb/genji/engine/badgerengine v0.12.0
	github.com/go-ocf/go-coap v0.0.0-20200224085725-3e22e8f506ea
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang/glog v0.0.0-20210429001901-424d2337a529 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/btree v1.0.1 // indirect
	github.com/google/flatbuffers v2.0.0+incompatible // indirect
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.1.2
	github.com/gorilla/websocket v1.4.0
	github.com/inconshreveable/log15 v0.0.0-20200109203555-b30bc20e4fd1 // indirect
	github.com/influxdata/influxdb-client-go/v2 v2.2.3
	github.com/jacobsa/go-serial v0.0.0-20180131005756-15cf729a72d4
	github.com/kevinburke/go-types v0.0.0-20200309064045-f2d4aea18a7a // indirect
	github.com/kevinburke/go.uuid v1.2.0 // indirect
	github.com/kevinburke/rest v0.0.0-20200429221318-0d2892b400f8 // indirect
	github.com/kevinburke/twilio-go v0.0.0-20200810163702-320748330fac
	github.com/klauspost/compress v1.12.1 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/nats-io/nats-server/v2 v2.2.2
	github.com/nats-io/nats.go v1.10.1-0.20210419223411-20527524c393
	github.com/ttacon/builder v0.0.0-20170518171403-c099f663e1c2 // indirect
	github.com/ttacon/libphonenumber v1.1.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.4 // indirect
	go.bug.st/serial v1.1.3
	go.etcd.io/bbolt v1.3.6
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20210603125802-9665404d3644 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.1.0 // indirect
	google.golang.org/protobuf v1.26.0
)

//replace github.com/nats-io/nats.go => github.com/cbrake/nats.go v1.10.1-0.20200817210920-7a8e05e18c84
//replace github.com/dgraph-io/badger/v3 v3.2011.1 => github.com/dgraph-io/badger/v3 v3.2012.0

go 1.16
