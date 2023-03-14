// serial encode test
package main

import (
	"fmt"
	"log"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
)

func main() {
	seq := byte(23)

	subject := ""

	points := data.Points{
		{Type: "temp", Value: 23},
		{Type: "description", Text: "temp sensor"},
	}

	data, err := client.SerialEncode(seq, subject, points)

	if err != nil {
		log.Fatal("Encode error: ", err)
	}

	fmt.Println("Encoded data: ", test.HexDump(data))

}
