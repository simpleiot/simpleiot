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
			data.NewPointString(data.PointTypeDescription, "0", "My Node"),
			data.NewPointFloat("tankLevel", "0", 12.523423423),
			data.NewPointFloat("current", "c0", 1.52323),
		},
	}

	res, err := renderNotifyTemplate(&device, `Alarm from {{.Description}}, tank level is {{printf "%.2f" (index .Ios "tankLevel")}}.`)

	if err != nil {
		t.Error("render failed: ", err)
	}

	if res != "Alarm from My Node, tank level is 12.52." {
		t.Error("rendered text is not correct: ", res)
	}

	fmt.Println("render result: ", res)
}
