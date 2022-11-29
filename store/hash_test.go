package store

import (
	"fmt"
	"sort"
	"testing"
)

func XorSum(a []int) int {
	var ret int
	for _, v := range a {
		ret = ret ^ v
	}

	return ret
}

func cloneArray[T any](a []T) []T {
	ret := make([]T, len(a))
	copy(ret, a)
	return ret
}

// We'd like a checksum that is commutative and can be updated incrementally
func TestXorChecksum(t *testing.T) {
	d1 := []int{2342000, 1928323, 29192, 41992, 29439}
	d2 := cloneArray(d1)
	sort.Ints(d2)
	// d1 and d2 slices now have save values, but in different order

	d1Sum := XorSum(d1)
	d2Sum := XorSum(d2)

	// check that checksum is commutative
	if d1Sum != d2Sum {
		t.Fatal("check sum is not commutative")
	}

	// simulate updating a value by replacing the value in the array
	// and checksumming the entire array, and also by incrementally
	// updating the checksum
	newValue := 101929
	d3 := cloneArray(d1)

	// replace a value in the array, and calculate the new checksum
	d3[4] = newValue
	d3Sum := XorSum(d3)

	// try to incrementally update the checksum by backing out the old
	// checksum value and adding the new value
	d3SumInc := d1Sum ^ d1[4] ^ newValue

	if d3Sum != d3SumInc {
		t.Fatal("Incremental checksum did not equal the complete checksum")
	}
}

// Test if we can compute XOR checksums in groups, and then XOR the groups
func TestXorChecksumGroup(t *testing.T) {
	d1 := []int{2342000, 1928323, 29192, 41992, 29439}

	d1Sum := XorSum(d1)

	d1SumA := XorSum(d1[:2])
	d1SumB := XorSum(d1[2:])

	d1SumAB := XorSum([]int{d1SumA, d1SumB})

	if d1Sum != d1SumAB {
		t.Fatal("Grouped checksum did not work")
		fmt.Printf("sums: %0x %0x %0x %0x\n", d1Sum, d1SumA, d1SumB, d1SumAB)
	}

	// it works, pretty neat!
}

func TestUpstreamHash(t *testing.T) {
	n := 12342342
	update := 22992343

	n1 := n ^ update

	updateCalc := n ^ n1

	if updateCalc != update {
		t.Fatal("Hmm, something went wrong")
	}

	// this is pretty neat as we can simply pass the update value upstream
	// and apply it to each node.
	// there is one disadvantage -- if the update travels through two paths, then
	// the update will cancel out once they merge again
}
