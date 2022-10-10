package store

import (
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
