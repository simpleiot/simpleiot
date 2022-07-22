package client

import (
	"fmt"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

func TestSerialEncodeDecode(t *testing.T) {
	seq := byte(123)
	subject := "test/subject/23"
	points := data.Points{
		{Type: data.PointTypeDescription, Text: "node description"},
		{Type: data.PointTypeValue, Value: 23.53},
	}

	d, err := SerialEncode(seq, subject, points)
	if err != nil {
		t.Fatal("Error encoding data: ", err)
	}

	seqD, subjectD, pointsD, err := SerialDecode(d)

	if err != nil {
		t.Error("Decode error: ", err)
	}

	if seq != seqD {
		t.Error("sequence mismatch")
	}

	if subject != subjectD {
		t.Error("subject mismatch")
	}

	if len(points) != len(pointsD) {
		t.Error("points len mismatch")
		fmt.Printf("points: %+v\n", points)
		fmt.Printf("pointsD: %+v\n", pointsD)
	}

	if points[0].Type != pointsD[0].Type {
		t.Error("points[0] description mismatch")
		fmt.Printf("points: %+v\n", points[0])
		fmt.Printf("pointsD: %+v\n", pointsD[0])
	}
}

func TestSerialEncodeDecodeNoContent(t *testing.T) {
	seq := byte(68)

	d, err := SerialEncode(seq, "", nil)
	if err != nil {
		t.Fatal("Error encoding: ", err)
	}

	seqD, _, _, err := SerialDecode(d)

	if err != nil {
		t.Error("Decode error: ", err)
	}

	if seq != seqD {
		t.Error("sequence mismatch")
	}
}
