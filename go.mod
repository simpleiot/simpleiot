module github.com/simpleiot/simpleiot

require (
	github.com/StephaneBunel/bresenham v0.0.0-20190213085234-b50c292e2054
	github.com/benbjohnson/genesis v0.2.1
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/gorilla/websocket v1.4.0
	github.com/mxmCherry/movavg v1.1.0
	github.com/onsi/ginkgo v1.7.0 // indirect
	github.com/onsi/gomega v1.4.3 // indirect
	github.com/pbnjay/pixfont v0.0.0-20190130005054-401bb7c6aee2
	github.com/timshannon/bolthold v0.0.0-20180829183128-83840edea944
	go.etcd.io/bbolt v1.3.0 // indirect
	golang.org/x/image v0.0.0-20190118043309-183bebdce1b2
	golang.org/x/sys v0.0.0-20190109145017-48ac38b7c8cb // indirect
	periph.io/x/periph v3.4.0+incompatible
)

replace periph.io/x/periph => github.com/cbrake/periph v3.4.991+incompatible

replace golang.org/x/image => github.com/cbrake/golang.org-x-image v0.0.1
