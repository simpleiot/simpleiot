package main

import (
	"log"
	"time"

	"github.com/genjidb/genji"
	"github.com/google/uuid"
)

// User type
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

	defer db.Close()

	err = db.Exec("CREATE TABLE users")
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

	err = db.Exec("INSERT INTO users VALUES ?", &u)
	if err != nil {
		log.Fatal("Error inserting user: ", err)
	}

	u = User{
		FirstName:   "Fred",
		LastName:    "Maple",
		PhoneNumber: "123-789-4562",
		Email:       "fred@admin.com",
	}

	count := 100
	start := time.Now()
	for i := 0; i < count; i++ {
		u.ID = uuid.New()
		err = db.Exec("INSERT INTO users VALUES ?", &u)
		if err != nil {
			log.Fatal("Error inserting user: ", err)
		}
	}

	log.Println("Insert time per record: ", time.Since(start)/time.Duration(count))

	log.Println("All done :-)")
}
