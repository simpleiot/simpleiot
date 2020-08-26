package main

import (
	"log"
	"time"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document"
)

// User type
type User struct {
	ID          int
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

	id := 0

	dataExists := false

	err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY);")
	if err != nil {
		if err != database.ErrTableAlreadyExists {
			log.Fatal("error creating users: ", err)
		}
	}

	err = db.Exec("CREATE INDEX idx_users_email ON users(email)")
	if err != nil {
		if err != database.ErrIndexAlreadyExists {
			log.Fatal("error creating index: ", err)
		}
		dataExists = true
	}

	if !dataExists {

		// insert first user, then a lot of another user
		u := User{
			ID:          id,
			FirstName:   "Joe",
			LastName:    "Oak",
			PhoneNumber: "123-456-7890",
			Email:       "joe@admin.com",
		}

		id++

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

		count := 100000
		start := time.Now()
		for i := 0; i < count; i++ {
			u.ID = id
			id++
			err = db.Exec("INSERT INTO users VALUES ?", &u)
			if err != nil {
				log.Fatal("Error inserting user: ", err)
			}
		}

		log.Println("Insert time per record: ", time.Since(start)/time.Duration(count))
	}

	func() {
		start := time.Now()

		res, err := db.Query("SELECT * FROM users WHERE email = ?", "joe@admin.com")

		if err != nil {
			log.Fatal("query error: ", err)
		}

		defer res.Close()

		count := 0

		err = res.Iterate(func(d document.Document) error {
			u := User{}
			err := document.StructScan(d, &u)
			if err != nil {
				log.Fatal("Error scanning document: ", err)
			}

			count++

			return nil
		})

		log.Printf("email query, documents found: %v, time: %v", count, time.Since(start))

	}()

	func() {
		start := time.Now()

		res, err := db.Query("SELECT * FROM users WHERE firstname = ?", "Joe")

		if err != nil {
			log.Fatal("query error: ", err)
		}

		defer res.Close()

		count := 0

		err = res.Iterate(func(d document.Document) error {
			u := User{}
			err := document.StructScan(d, &u)
			if err != nil {
				log.Fatal("Error scanning document: ", err)
			}

			count++

			return nil
		})

		log.Printf("firstname query, documents found: %v, time: %v", count, time.Since(start))

	}()

	log.Println("All done :-)")
}
