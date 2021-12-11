package time_wheel

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"
	"time-wheel/conf"
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
	t.timeWheel = NewTimeWheel(0, int64(10*time.Millisecond), conf.Conf.WheelConfig.Size, time.Now().Unix(), 0, make(chan *NodeList, conf.Conf.WheelConfig.Size))
	t.receiveChan = make(chan *TimeNode, 1024)
	t.cmdChan = make(chan cmd, 1024)
	t.quitChan = make(chan struct{}, 1)
	t.tick = ticker
	go t.timeWheel.ProcMsg(ticker, t.cmdChan, t.receiveChan, t.quitChan)
}

func (t *Timer) stop() {
	t.tick.Stop()
	close(t.quitChan)

}
func BuildTimeNode(triggerType TimerType, duration int64) (*TimeNode, error) {
	//计算到期时间
	expireTime := time.Now().Unix() + duration
	nodeId, err := buildNodeId()
	if err != nil {
		return nil, err
	}
	node := &TimeNode{
		DelayTime:  duration,
		SignalChan: make(chan struct{}, 1),
		TimerType:  triggerType,
		ExpireTime: expireTime,
		NodeId:     nodeId,
	}
	return node, err
}

func AddNode(node *TimeNode) (chan struct{}, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	select {
	case timer.receiveChan <- node:
		return node.SignalChan, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("")
	}
}

func ConvertUnix(data interface{}) (int64, error) {
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

/*
func AfterTimer(d time.Duration) (chan struct{}, error) {

	duration, err := ConvertUnix(d)
	if err != nil {
		return nil, err
	}
	node := &TimeNode{DelayTime: duration, SignalChan: make(chan struct{}, 1), TimerType: DelayTimerNode}
	return AddNode(node)

}

func AfterDurationStr(s string) (chan struct{}, error) {
	duration, err := ConvertUnix(s)
	if err != nil {
		return nil, err
	}
	node, err := BuildTimeNode(DelayTimerNode, duration)
	if err != nil {
		return nil, err
	}
	return AddNode(node)
}

//针对Ticker型时钟，后续可扩展支持每天几点，每周的几点触发
func TickTimer(d time.Duration) (chan struct{}, error) {
	duration := int64(d)
	if duration <= 0 {
		//报错
		return nil, fmt.Errorf("时间转换错误")
	}
	node, err := BuildTimeNode(TickTimerNode, duration)
	if err != nil {
		return nil, err
	}
	node.RefreshHandler = &InternalCycle{DelayTime: node.DelayTime}
	node.RefreshHandler.Refresh()
	return AddNode(node)
}

func TickTimerStr(s string, opt []time.Weekday) (chan struct{}, error) {
	_, err := time.ParseInLocation(" 15:04:05", s, time.Local)
	if err != nil {
		return nil, err
	}

	node, err := BuildTimeNode(TickTimerNode, 0)
	if err != nil {
		return nil, err
	}
	node.RefreshHandler = &UnixCycle{ExpireStr: s, Opt: opt}
	node.RefreshHandler.Refresh()
	return AddNode(node)
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
*/
