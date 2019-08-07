package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/timshannon/badgerhold"
	"github.com/timshannon/bolthold"
)

// Sample1 only contains Value
type Sample1 struct {
	Time  time.Time
	Value float64
}

// Sample2 contains value + min/max
type Sample2 struct {
	Time  time.Time
	Value float64
	Min   float64
	Max   float64
}

func main() {
	os.RemoveAll("./temp")

	err := os.Mkdir("./temp", os.ModePerm)
	if err != nil {
		log.Fatal("Error creating directory: ", err)
	}

	bolt1, err := bolthold.Open("temp/bolt1.db", 0666, nil)
	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	bolt2, err := bolthold.Open("temp/bolt2.db", 0666, nil)
	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	options := badgerhold.DefaultOptions
	options.Dir = "./temp/badger1"
	options.ValueDir = "./temp/badger1"

	badger1, err := badgerhold.Open(options)
	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	options.Dir = "./temp/badger2"
	options.ValueDir = "./temp/badger2"

	badger2, err := badgerhold.Open(options)
	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	gob1, err := os.Create("temp/gob1.dat")
	if err != nil {
		log.Fatal("Error opening gob1: ", err)
	}

	gob1E := gob.NewEncoder(gob1)

	gob2, err := os.Create("temp/gob2.dat")
	if err != nil {
		log.Fatal("Error opening gob2: ", err)
	}

	gob2E := gob.NewEncoder(gob2)

	for i := 0; i < 1000; i++ {
		sample1 := Sample1{
			Value: rand.Float64(),
		}

		sample2 := Sample2{
			Value: sample1.Value,
			Min:   rand.Float64(),
			Max:   rand.Float64(),
		}

		gob1E.Encode(sample1)
		gob2E.Encode(sample2)

		err := bolt1.Insert(time.Now(), sample1)
		if err != nil {
			log.Println("Error inserting value")
		}

		err = bolt2.Insert(time.Now(), sample2)
		if err != nil {
			log.Println("Error inserting value")
		}

		err = badger1.Insert(time.Now(), sample1)
		if err != nil {
			log.Println("Error inserting value")
		}

		err = badger2.Insert(time.Now(), sample2)
		if err != nil {
			log.Println("Error inserting value")
		}

	}

	bolt1.Close()
	bolt2.Close()
	badger1.Close()
	badger2.Close()
	gob1.Close()
	gob2.Close()

	// get stats on files
	files := []string{"gob1.dat", "gob2.dat", "bolt1.db", "bolt2.db"}

	for _, f := range files {
		fO, err := os.Open("temp/" + f)
		if err != nil {
			log.Fatal("Could not open file to get stats: ", err)
		}

		stat, err := fO.Stat()
		if err != nil {
			log.Fatal("Could not get stats: ", err)
		}

		fmt.Printf("%v: size: %v\n", f, stat.Size())
	}
}
