package main

import (
	"log"

	"github.com/genjidb/genji"
	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	FirstName   string
	LastName    string
	PhoneNumber string
	Email       string
}

func main() {
	db, err := genji.Open("genji-test.db")

	if err != nil {

		log.Fatal(err)
	}

	err = db.Exec("CREATE TABLE users id ")
	if err != nil {
		log.Fatal("error creating users: ", err)
	}

	err = db.Exec("CREATE INDEX idx_users_email ON users(email)")
	if err != nil {
		log.Fatal("error creating index: ", err)
	}

	// insert first user, then a lot of another user
	u := User{
		ID:          uuid.New(),
		FirstName:   "Joe",
		LastName:    "Oak",
		PhoneNumber: "123-456-7890",
		Email:       "joe@admin.com",
	}

	for {

	}
}
