module github.com/simpleiot/simpleiot

require (
	github.com/360EntSecGroup-Skylar/excelize/v2 v2.0.2
	github.com/RobinUS2/golang-moving-average v0.0.0-20190414143424-55c2d531d53f
	github.com/StephaneBunel/bresenham v0.0.0-20190213085234-b50c292e2054
	github.com/adrianmo/go-nmea v1.1.1-0.20190321164421-7572fbeb90aa
	github.com/beevik/ntp v0.3.0
	github.com/benbjohnson/genesis v0.2.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/cbrake/go-serial v0.0.0-20171213223811-0cd42b853914
	github.com/cbrake/influxdbhelper/v2 v2.1.4
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/donovanhide/eventsource v0.0.0-20171031113327-3ed64d21fb0b
	github.com/go-ocf/go-coap v0.0.0-20200224085725-3e22e8f506ea
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.0
	github.com/inconshreveable/log15 v0.0.0-20200109203555-b30bc20e4fd1 // indirect
	github.com/influxdata/influxdb1-client v0.0.0-20190809212627-fc22c7df067e
	github.com/jacobsa/go-serial v0.0.0-20180131005756-15cf729a72d4
	github.com/kevinburke/go-types v0.0.0-20200309064045-f2d4aea18a7a // indirect
	github.com/kevinburke/go.uuid v1.2.0 // indirect
	github.com/kevinburke/rest v0.0.0-20200429221318-0d2892b400f8 // indirect
	github.com/kevinburke/twilio-go v0.0.0-20200804051954-bc580f944739
	github.com/kr/pretty v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mitchellh/go-ps v1.0.0
	github.com/mxmCherry/movavg v1.1.0
	github.com/onsi/ginkgo v1.7.0 // indirect
	github.com/onsi/gomega v1.4.3 // indirect
	github.com/pbnjay/pixfont v0.0.0-20190130005054-401bb7c6aee2
	github.com/pkg/errors v0.8.1
	github.com/svent/go-nbreader v0.0.0-20150201200112-7cef48da76dc
	github.com/timshannon/badgerhold v0.0.0-20190415130923-192650dd187a
	github.com/timshannon/bolthold v0.0.0-20200420150217-0e8c0be6fd3c
	github.com/ttacon/builder v0.0.0-20170518171403-c099f663e1c2 // indirect
	github.com/ttacon/libphonenumber v1.1.0 // indirect
	go.bug.st/serial v1.1.0
	go.etcd.io/bbolt v1.3.4
	golang.org/x/image v0.0.0-20190910094157-69e4b8554b2a
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b
	golang.org/x/sys v0.0.0-20200413165638-669c56c373c4 // indirect
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	periph.io/x/periph v3.4.0+incompatible
)

replace periph.io/x/periph => github.com/cbrake/periph v3.4.991+incompatible

replace golang.org/x/image => github.com/cbrake/golang.org-x-image v0.0.1

go 1.13
