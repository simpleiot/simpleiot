package main

import (
	"context"
	"fmt"
	"log"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
)

type User struct {
	ID        string `json:"id" boltholdKey:"ID"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Pass      string `json:"pass"`
}

// returns true if db already exists
func setup(db *genji.DB) {
	ctx := context.Background()
	err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS users;")
	if err != nil {
		log.Fatal("error creating users: ", err)
	}

}

func main() {
	//db, err := genji.Open(":memory:")
	db, err := genji.Open("genji-test4.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	setup(db)

	u1 := User{ID: "01", FirstName: "cliff", LastName: "brake", Phone: "",
		Email: "admin@admin.com", Pass: "admin"}

	ctx := context.Background()

	err = db.Exec(ctx, `insert into users values ?`, u1)
	if err != nil {
		log.Fatal("Error inserting user: ", err)
	}

	doc, err := db.QueryDocument(ctx, `select * from users`)
	if err != nil {
		log.Fatal("Query error: ", err)
	}

	var u2 User
	err = document.StructScan(doc, &u2)
	if err != nil {
		log.Fatal("ScanStruct error: ", err)
	}

	fmt.Println("u2: ", u2)
}
