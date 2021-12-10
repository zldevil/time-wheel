package time_wheel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestWheel(t *testing.T) {
	timeWheel := NewTimeWheel(0, int64(1*time.Second), 10, time.Now().Unix(), 0, make(chan *NodeList, 10))
	cur := time.Now().Unix()
	go timeWheel.Start()
	node := &TimeNode{signalChan: make(chan struct{}, 1)}
	timeWheel.addTimerNode(node)
	s := fmt.Sprint(node)
	t.Log(s)
	for {
		select {
		case <-node.signalChan:
			cur1 := time.Now().Unix()
			sub := cur1 - cur
			t.Log(sub)
		}
	}
	t.Log("1111")
}

func TestAddq(t *testing.T) {
	s := sync.WaitGroup{}
	dataChan := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case e, ok := <-dataChan:
				t.Log(e)
				t.Log(ok)
			}
		}
	}()
	close(dataChan)

	s.Add(1)
	s.Wait()
	t.Log("")
}
