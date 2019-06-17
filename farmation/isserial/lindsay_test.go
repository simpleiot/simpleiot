package isserial

import (
	"bytes"
	"fmt"
	"testing"
)

func TestLindsay(t *testing.T) {
	dataReader := bytes.NewBuffer(LindsayTestData)

	lindsay := NewLindsay(dataReader)

	count := 0

	for {
		status, err := lindsay.Read()
		if err == errorNotLindsayStatus {
			continue
		}

		if err != nil {
			fmt.Println("TestLindsay error: ", err)
			break
		}

		count++
		fmt.Printf("Lindsay status: %+v\n", status)

		if status.PosWithOffset != 34 {
			t.Error("PosWithOffset error")
		}

		if status.PosWithoutOffset != 99 {
			t.Error("PosWithoutOffset error")
		}

		if status.Status != 0x200 {
			t.Error("Status error")
		}

		if status.Pressure != 5 {
			t.Error("pressure error")
		}

		if status.State != 0 {
			t.Error("state error")
		}
	}

	if count != 1 {
		t.Error("Expected on Lindsay packet")
	}
}
