package api

import (
	"fmt"
	"time"
	"time-wheel/time_wheel"
)

func AfterTimer(d time.Duration) (chan struct{}, error) {

	duration, err := time_wheel.ConvertUnix(d)
	if err != nil {
		return nil, err
	}
	node := &time_wheel.TimeNode{DelayTime: duration, SignalChan: make(chan struct{}, 1), TimerType: time_wheel.DelayTimerNode}
	return time_wheel.AddNode(node)

}

func AfterDurationStr(s string) (chan struct{}, error) {
	duration, err := time_wheel.ConvertUnix(s)
	if err != nil {
		return nil, err
	}
	node, err := time_wheel.BuildTimeNode(time_wheel.DelayTimerNode, duration)
	if err != nil {
		return nil, err
	}
	return time_wheel.AddNode(node)
}

//针对Ticker型时钟，后续可扩展支持每天几点，每周的几点触发
func TickTimer(d time.Duration) (chan struct{}, error) {
	duration := int64(d)
	if duration <= 0 {
		//报错
		return nil, fmt.Errorf("时间转换错误")
	}
	node, err := time_wheel.BuildTimeNode(time_wheel.TickTimerNode, duration)
	if err != nil {
		return nil, err
	}
	node.RefreshHandler = &time_wheel.InternalCycle{DelayTime: node.DelayTime}
	node.RefreshHandler.Refresh()
	return time_wheel.AddNode(node)
}

func TickTimerStr(s string, opt []time.Weekday) (chan struct{}, error) {
	_, err := time.ParseInLocation(" 15:04:05", s, time.Local)
	if err != nil {
		return nil, err
	}

	node, err := time_wheel.BuildTimeNode(time_wheel.TickTimerNode, 0)
	if err != nil {
		return nil, err
	}
	node.RefreshHandler = &time_wheel.UnixCycle{ExpireStr: s, Opt: opt}
	node.RefreshHandler.Refresh()
	return time_wheel.AddNode(node)
}
func RemoveNode(nodeId uint64) {
	/*cmd := &time_wheel.removeNodeCmd{nodeId: nodeId}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	select {
	case time_wheel.timer.cmdChan <- cmd:
	case <-ctx.Done():
		return
	}*/
}
