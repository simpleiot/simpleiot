module github.com/simpleiot/simpleiot

require (
	github.com/benbjohnson/genesis v0.2.1
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/gorilla/websocket v1.4.0
	github.com/pbnjay/pixfont v0.0.0-20171109033744-80412ecf517a
	github.com/timshannon/bolthold v0.0.0-20180829183128-83840edea944
	go.etcd.io/bbolt v1.3.0 // indirect
	golang.org/x/image v0.0.0-20190118043309-183bebdce1b2
	golang.org/x/sys v0.0.0-20190109145017-48ac38b7c8cb // indirect
	periph.io/x/periph v3.4.0+incompatible
)

replace periph.io/x/periph => github.com/cbrake/periph v3.4.991+incompatible

replace golang.org/x/image => github.com/cbrake/golang.org-x-image v0.0.1
