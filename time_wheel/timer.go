package time_wheel

import (
	"context"
	"fmt"
	"time"
)

var (
	timeWheel   *TimeWheel
	receiveChan chan *TimeNode
)

func Init(tickMs int64, wheelSize int64) {
	timeWheel = NewTimeWheel(0, tickMs, wheelSize, time.Now().Unix(), 0, make(chan *TimeNodeList, wheelSize))
	receiveChan = make(chan *TimeNode, 1024)
	timeWheel.Start()
}

func AfterTimer(d time.Duration) (chan struct{}, error) {
	duration := int64(d)
	if duration <= 0 {
		//报错
		return nil, fmt.Errorf("时间转换错误")
	}
	node := &TimeNode{delayTime: duration, signalChan: make(chan struct{}, 1), timerType: DelayTimerNode}
	return addNode(node)

	//return timeWheel.afterTime(duration)
}

func TickTimer(d time.Duration) (chan struct{}, error) {
	duration := int64(d)
	if duration <= 0 {
		//报错
		return nil, fmt.Errorf("时间转换错误")
	}
	node := &TimeNode{delayTime: duration, signalChan: make(chan struct{}, 1), timerType: TickTimerNode}
	return addNode(node)
}

func addNode(node *TimeNode) (chan struct{}, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	select {
	case receiveChan <- node:
		return node.signalChan, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("")
	}
}
