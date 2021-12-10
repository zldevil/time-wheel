package time_wheel

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"
)

const (
	FormatDateTime = "2006-01-02 15:04:05"
	FormatDate     = "2006-01-02"
)

type Timer struct {
	timeWheel   *TimeWheel
	receiveChan chan *TimeNode
	cmdChan     chan cmd
	quitChan    chan struct{}
	exit        bool
	tick        *time.Ticker
}

var timer *Timer

func init() {
	timer = &Timer{}
}

func (t *Timer) Start() {
	//size和tickMs从配置文件中读取
	ticker := time.NewTicker(10 * time.Millisecond)
	t.timeWheel = NewTimeWheel(0, int64(10*time.Millisecond), Conf.WheelConfig.Size, time.Now().Unix(), 0, make(chan *NodeList, Conf.WheelConfig.Size))
	t.receiveChan = make(chan *TimeNode, 1024)
	t.cmdChan = make(chan cmd, 1024)
	t.quitChan = make(chan struct{}, 1)
	t.tick = ticker
	go t.timeWheel.Start(ticker, t.cmdChan, t.receiveChan, t.quitChan)
}

func (t *Timer) stop() {
	t.tick.Stop()

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
	node, err := buildTimeNode(DelayTimerNode, duration)
	if err != nil {
		return nil, err
	}
	return addNode(node)
}

//针对Ticker型时钟，后续可扩展支持每天几点，每周的几点触发
func TickTimer(d time.Duration) (chan struct{}, error) {
	duration := int64(d)
	if duration <= 0 {
		//报错
		return nil, fmt.Errorf("时间转换错误")
	}
	node, err := buildTimeNode(TickTimerNode, duration)
	if err != nil {
		return nil, err
	}
	node.refreshHandler = &internalCycle{delayTime: node.delayTime}
	node.refreshHandler.Refresh()
	return addNode(node)
}

func TickTimerStr(s string, opt []time.Weekday) (chan struct{}, error) {
	_, err := time.ParseInLocation(" 15:04:05", s, time.Local)
	if err != nil {
		return nil, err
	}

	node, err := buildTimeNode(TickTimerNode, 0)
	if err != nil {
		return nil, err
	}
	node.refreshHandler = &unixCycle{expireStr: s, opt: opt}
	node.refreshHandler.Refresh()
	return addNode(node)
}
func RemoveNode(nodeId uint64) {
	cmd := &removeNodeCmd{nodeId: nodeId}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	select {
	case timer.cmdChan <- cmd:
	case <-ctx.Done():
		return
	}
}

func buildTimeNode(triggerType TimerType, duration int64) (*TimeNode, error) {
	//计算到期时间
	expireTime := time.Now().Unix() + duration
	nodeId, err := buildNodeId()
	if err != nil {
		return nil, err
	}
	node := &TimeNode{
		delayTime:  duration,
		signalChan: make(chan struct{}, 1),
		timerType:  triggerType,
		expireTime: expireTime,
		NodeId:     nodeId,
	}
	return node, err
}

func addNode(node *TimeNode) (chan struct{}, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	select {
	case timer.receiveChan <- node:
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
