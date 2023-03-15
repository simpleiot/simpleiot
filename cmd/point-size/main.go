// test size of point encoding
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"google.golang.org/protobuf/proto"
)

func main() {
	sizeEncodedPoint := func(p data.Point) int {
		pE, err := p.ToPb()

		if err != nil {
			log.Fatal("Error encoding: ", err)
		}

		buf, err := proto.Marshal(&pE)

		if err != nil {
			log.Fatal("Marshal error: ", err)
		}

		return len(buf)
	}

	p := data.Point{Type: "p"}

	fmt.Printf("Simple: %+v -> %v bytes\n", p, sizeEncodedPoint(p))

	buf, _ := (&data.Points{p}).ToPb()

	fmt.Printf("Add array: %+v -> %v bytes\n", p, len(buf))

	p = data.Point{Time: time.Now(), Type: "value", Value: 232.32}
	fmt.Printf("Typical point: %+v -> %v bytes\n", p, sizeEncodedPoint(p))

	// 10 typical points in an array
	var pArray data.Points

	for i := 0; i < 10; i++ {
		pArray = append(pArray, p)
	}

	pArrayBuf, _ := pArray.ToPb()
	pArrayLen := len(pArrayBuf)
	fmt.Printf("Size of 10 typical points: %v, per point: %v\n", pArrayLen, float64(pArrayLen)/10)

}

// This program outputs:
// Simple: T:p V:0.000 0001-01-01T00:00:00Z -> 16 bytes
// Add array: T:p V:0.000 0001-01-01T00:00:00Z -> 18 bytes
// Typical point: T:value V:232.320 2022-10-04T14:57:34-04:00 -> 26 bytes
// Size of 10 typical points: 280, per point: 28
