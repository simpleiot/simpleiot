package client

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

type testClient struct {
	stop     chan struct{}
	stopOnce sync.Once
}

func newTestClient() *testClient {
	return &testClient{stop: make(chan struct{})}
}

func (tc *testClient) Run() error {
	<-tc.stop
	return errors.New("client stopped")
}

func (tc *testClient) Stop(_ error) {
	tc.stopOnce.Do(func() { close(tc.stop) })
}

func TestGroup(_ *testing.T) {
	g := NewRunGroup("testGroup")
	testC := newTestClient()
	g.Add(testC)

	groupErr := make(chan error)

	// first try to stop everything by stopping client
	go func() {
		groupErr <- g.Run()
	}()

	testC.Stop(nil)
	fmt.Println("group returned: ", <-groupErr)

	// now try stopping everything by stopping group
	go func() {
		groupErr <- g.Run()
	}()

	g.Stop(nil)
	fmt.Println("group returned: ", <-groupErr)
}
