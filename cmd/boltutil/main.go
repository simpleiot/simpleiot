package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	bolt "go.etcd.io/bbolt"
)

func dumpBboltKeys(filepath string) {
	db, err := bolt.Open(filepath, 0666, nil)
	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	defer db.Close()

	db.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			fmt.Println("Bucket: ", string(name))
			b.ForEach(func(k, v []byte) error {
				fmt.Printf("   key=%v, value=%s\n", hex.Dump(k), v)
				return nil
			})
			return nil
		})
		return nil
	})
}

func main() {
	if len(os.Args) < 2 {
		log.Println("Usage boltutil <filename>")
		os.Exit(-1)
	}
	dumpBboltKeys(os.Args[1])
}
