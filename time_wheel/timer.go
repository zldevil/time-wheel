package time_wheel

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"
)

var (
	timeWheel   *TimeWheel
	receiveChan chan *TimeNode
)

const (
	FormatDateTime = "2006-01-02 15:04:05"
)

func Init(tickMs int64, wheelSize int64) {
	timeWheel = NewTimeWheel(0, tickMs, wheelSize, time.Now().Unix(), 0, make(chan *TimeNodeList, wheelSize))
	receiveChan = make(chan *TimeNode, 1024)
	timeWheel.Start()
}

func AfterTimer(d time.Duration) (chan struct{}, error) {

	duration, err := convertUnix(d)
	if err != nil {
		return nil, err
	}
	node := &TimeNode{delayTime: duration, signalChan: make(chan struct{}, 1), timerType: DelayTimerNode}
	return addNode(node)

}

func AfterDurationStr(s string) (chan struct{}, error) {
	duration, err := convertUnix(s)
	if err != nil {
		return nil, err
	}
	node := buildTimeNode(DelayTimerNode, duration)
	return addNode(node)
}

//针对Ticker型时钟，后续可扩展支持每天几点，每周的几点触发
func TickTimer(d time.Duration) (chan struct{}, error) {
	duration := int64(d)
	if duration <= 0 {
		//报错
		return nil, fmt.Errorf("时间转换错误")
	}
	node := buildTimeNode(TickTimerNode, duration)
	return addNode(node)
}

func buildTimeNode(triggerType TimerType, duration int64) *TimeNode {
	//计算到期时间
	expireTime := time.Now().Unix() + duration
	node := &TimeNode{
		delayTime:  duration,
		signalChan: make(chan struct{}, 1),
		timerType:  triggerType,
		expireTime: expireTime,
	}
	return node
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

func convertUnix(data interface{}) (int64, error) {
	var (
		duration int64
		err      error
	)
	switch reflect.TypeOf(data).Kind() {
	case reflect.Int64:
		d, ok := data.(time.Duration)
		if !ok {
			return 0, nil
		}
		duration = int64(d)
	case reflect.String:
		dStr := data.(string)
		if d, err := time.ParseDuration(dStr); err == nil {
			duration = int64(d)
			break
		}
		t, err := time.Parse(FormatDateTime, dStr)
		if err != nil {
			duration = -1
			err = errors.New("time param is incorrect")
			break
		}
		duration = time.Now().Unix() - t.Unix()
	default:
		duration = -1
		err = errors.New("time param type is incorrect")
	}
	if duration <= 0 {
		err = fmt.Errorf("time param is incorrect")
	}
	return duration, err
}
