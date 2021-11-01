package node

import (
	"fmt"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

func TestNotifyTemplate(t *testing.T) {
	device := data.Node{
		ID: "1234",
		Points: []data.Point{
			{
				Type: data.PointTypeDescription,
				Text: "My Node",
			},
			{
				Type:  "tankLevel",
				Key:   "",
				Value: 12.523423423,
			},
			{
				Type:  "current",
				Key:   "c0",
				Value: 1.52323,
			},
		},
	}

	res, err := renderNotifyTemplate(&device, `Alarm from {{.Description}}, tank level is {{printf "%.2f" (index .Ios "tankLevel")}}.`)

	if err != nil {
		t.Error("render failed: ", err)
	}

	if res != "Alarm from My Node, tank level is 12.52." {
		t.Error("rendered text is not correct")
	}

	fmt.Println("render result: ", res)
}
