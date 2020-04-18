module github.com/simpleiot/simpleiot

require (
	github.com/360EntSecGroup-Skylar/excelize/v2 v2.0.2
	github.com/RobinUS2/golang-moving-average v0.0.0-20190414143424-55c2d531d53f
	github.com/StephaneBunel/bresenham v0.0.0-20190213085234-b50c292e2054
	github.com/adrianmo/go-nmea v1.1.1-0.20190321164421-7572fbeb90aa
	github.com/beevik/ntp v0.2.0
	github.com/benbjohnson/genesis v0.2.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/cbrake/go-serial v0.0.0-20171213223811-0cd42b853914
	github.com/cbrake/influxdbhelper/v2 v2.1.4
	github.com/davecgh/go-spew v1.1.1
	github.com/donovanhide/eventsource v0.0.0-20171031113327-3ed64d21fb0b
	github.com/go-ocf/go-coap v0.0.0-20200110075231-40e35a4cacdf
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/gorilla/websocket v1.4.0
	github.com/influxdata/influxdb1-client v0.0.0-20190809212627-fc22c7df067e
	github.com/jacobsa/go-serial v0.0.0-20180131005756-15cf729a72d4
	github.com/kr/pretty v0.1.0 // indirect
	github.com/mitchellh/go-ps v1.0.0
	github.com/mxmCherry/movavg v1.1.0
	github.com/onsi/ginkgo v1.7.0 // indirect
	github.com/onsi/gomega v1.4.3 // indirect
	github.com/pbnjay/pixfont v0.0.0-20190130005054-401bb7c6aee2
	github.com/pkg/errors v0.8.1
	github.com/svent/go-nbreader v0.0.0-20150201200112-7cef48da76dc
	github.com/timshannon/badgerhold v0.0.0-20190415130923-192650dd187a
	github.com/timshannon/bolthold v0.0.0-20200316231344-dc30e2b2f90c
	go.etcd.io/bbolt v1.3.3
	golang.org/x/image v0.0.0-20190910094157-69e4b8554b2a
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
	golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	periph.io/x/periph v3.4.0+incompatible
)

replace periph.io/x/periph => github.com/cbrake/periph v3.4.991+incompatible

replace golang.org/x/image => github.com/cbrake/golang.org-x-image v0.0.1

go 1.13
