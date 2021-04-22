package main

import (
	"fmt"
	"log"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
)

// Test data value
type Test struct {
	ID    string
	Value string
}

func main() {
	dbInst, err := genji.NewDb("bolt", "./")
	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	rootID := dbInst.RootNodeID()

	rootNode, err := dbInst.Node(rootID)

	if err != nil {
		log.Fatal("Error getting root node: ", err)
	}

	fmt.Printf("Root node: %+v\n", rootNode)

	err = dbInst.NodePoint(rootID, data.Point{
		Type: data.PointTypeDescription,
		Text: "node #1",
	})

	if err != nil {
		log.Fatal("Error updating node: ", err)
	}

	log.Println("All done :-)")
}
