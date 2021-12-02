package time_wheel

type TimerType uint32

const (
	DelayTimerNode TimerType = iota
	TickTimerNode
)

//对象池？
type TimeNode struct {
	delayTime  int64
	expireTime int64
	signalChan chan struct{}
	timerType  TimerType //定时任务还是延迟任务
}

type TimeNodeList struct {
	TimerNodeList []*TimeNode //后续可能改成链表结构
}

//封装链表方法
