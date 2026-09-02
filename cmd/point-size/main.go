// test size of point encoding
package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

func main() {
	sizeEncodedPoint := func(p data.Point) int {
		buf := &bytes.Buffer{}
		p.Encode(buf)
		return buf.Len()
	}

	p := data.Point{Type: "p"}

	fmt.Printf("Simple: %+v -> %v bytes\n", p, sizeEncodedPoint(p))

	buf := (&data.Points{p}).Encode()

	fmt.Printf("Add array: %+v -> %v bytes\n", p, len(buf))

	p = data.NewPointFloat("value", "", 232.32)
	p.Time = time.Now()
	fmt.Printf("Typical point: %+v -> %v bytes\n", p, sizeEncodedPoint(p))

	// 10 typical points in an array
	var pArray data.Points

	for i := 0; i < 10; i++ {
		pArray = append(pArray, p)
	}

	pArrayBuf := pArray.Encode()
	pArrayLen := len(pArrayBuf)
	fmt.Printf("Size of 10 typical points: %v, per point: %v\n", pArrayLen, float64(pArrayLen)/10)

	nodes := data.Nodes{{ID: "node-id", Type: "device", Parent: "parent-id", Points: pArray}}
	fmt.Printf("Node with 10 typical points: %v bytes\n", len(data.EncodeNodes(nodes, nil)))
}

// This program outputs:
// Simple: T:p V:0.000  -> 22 bytes
// Add array: T:p V:0.000  -> 26 bytes
// Typical point: T:value V:232.320 2026-09-01T15:35:27-04:00 -> 34 bytes
// Size of 10 typical points: 344, per point: 34.4
// Node with 10 typical points: 383 bytes
