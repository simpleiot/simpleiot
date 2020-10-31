package main

import (
	"fmt"
	"log"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	"github.com/simpleiot/simpleiot/data"
)

func main() {
	//db, err := genji.Open(":memory:")
	db, err := genji.Open("data.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	doc, err := db.QueryDocument(`select * from users where email = 'admin@admin.com' and pass = 'admin'`)
	if err != nil {
		log.Fatal("Query error: ", err)
	}

	var u2 data.User
	err = document.StructScan(doc, &u2)
	if err != nil {
		log.Fatal("ScanStruct error: ", err)
	}

	fmt.Println("u2: ", u2)
}
