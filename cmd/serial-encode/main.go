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
		data.NewPointFloat("temp", "", 23),
		data.NewPointString("description", "", "temp sensor"),
	}

	data, err := client.SerialEncode(seq, subject, points)

	if err != nil {
		log.Fatal("Encode error: ", err)
	}

	fmt.Println("Encoded data: ", test.HexDump(data))

}
