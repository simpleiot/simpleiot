package isflow

import (
	"fmt"
	"testing"

	movingaverage "github.com/RobinUS2/golang-moving-average"
)

func TestMovingAverage(t *testing.T) {
	ma := movingaverage.New(30)

	ma.Add(5)

	if ma.Avg() != 5 {
		fmt.Println("average with one point != 5", ma.Avg())
	}

	ma.Add(5)

	if ma.Avg() != 5 {
		fmt.Println("average with two points != 5", ma.Avg())
	}
}
