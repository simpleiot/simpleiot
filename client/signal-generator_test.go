package client_test

import (
	"math"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestSignalGeneratorResumes disables a running generator and enables it
// again. The signal continues from where it stopped rather than starting
// over from the initial value.
func TestSignalGeneratorResumes(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	// a triangle wave rising 0.8 a sample, 20 samples a second, from 10
	sg := client.SignalGenerator{
		ID:          "ID-sg",
		Parent:      root.ID,
		Description: "ramp",
		SignalType:  "triangle",
		MinValue:    10,
		MaxValue:    90,
		Frequency:   0.1,
		SampleRate:  20,
	}

	err = client.SendNodeType(nc, sg, "test")
	if err != nil {
		t.Fatal("Error sending generator node: ", err)
	}

	get, stopWatch, err := client.NodeWatcher[client.SignalGenerator](nc, sg.ID, sg.Parent)
	if err != nil {
		t.Fatal("Error watching generator: ", err)
	}
	defer stopWatch()

	waitFor := func(cond func() bool, msg string) {
		start := time.Now()
		for !cond() {
			if time.Since(start) > 3*time.Second {
				t.Fatal(msg)
			}
			<-time.After(10 * time.Millisecond)
		}
	}

	waitFor(func() bool { return get().Value > 25 }, "generator did not ramp")

	send := func(disabled float64) {
		p := data.NewPointFloat(data.PointTypeDisabled, "", disabled)
		p.Origin = "test"
		err := client.SendNodePoint(nc, sg.ID, p, true)
		if err != nil {
			t.Fatal("Error sending point: ", err)
		}
	}

	send(1)
	time.Sleep(200 * time.Millisecond)
	before := get().Value
	time.Sleep(300 * time.Millisecond)
	if got := get().Value; got != before {
		t.Fatalf("generator ran while disabled: %v then %v", before, got)
	}

	send(0)
	waitFor(func() bool { return get().Value != before }, "generator did not resume")
	after := get().Value
	if math.Abs(after-before) > 5 {
		t.Fatalf("generator started over: %v before, %v after", before, after)
	}
}
