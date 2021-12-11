package time_wheel

import (
	"fmt"
	"github.com/sony/sonyflake"
	"log"
	"sort"
	"sync"
	"time"
)

type TimerType uint32

const (
	DelayTimerNode TimerType = iota
	TickTimerNode
)

const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
	secondsPerWeek   = 7 * secondsPerDay
	daysPer400Years  = 365*400 + 97
	daysPer100Years  = 365*100 + 24
	daysPer4Years    = 365*4 + 1
)

type TimeNode struct {
	DelayTime      int64
	ExpireTime     int64
	SignalChan     chan struct{}
	NodeId         uint64
	TimerType      TimerType
	RefreshHandler NodeHandler
	next           *TimeNode
	prev           *TimeNode
	nodeList       *NodeList
}

func buildNodeId() (uint64, error) {
	flake := sonyflake.NewSonyflake(sonyflake.Settings{})
	id, err := flake.NextID()
	if err != nil {
		log.Fatalf("flake.NextID() failed with %s\n", err)
	}
	// Note: this is base16, could shorten by encoding as base62 string
	fmt.Printf("build NodeId :  %x\n", id)
	return id, err
}

type NodeHandler interface {
	Refresh() (int64, int64)
}

type InternalCycle struct {
	DelayTime int64
}

func (i *InternalCycle) Refresh() (int64, int64) {
	expireTime := time.Now().Unix() + i.DelayTime
	return expireTime, i.DelayTime
}

type UnixCycle struct {
	ExpireTimes []int64
	Opt         []time.Weekday
	ExpireStr   string
}

func (u *UnixCycle) Refresh() (int64, int64) {
	var (
		timeNow    = time.Now()
		delayTime  int64
		expireTime int64
	)

	for i := 0; i < len(u.ExpireTimes); i++ {
		if timeNow.Unix() < u.ExpireTimes[i] {
			expireTime = u.ExpireTimes[i]
			delayTime = expireTime - timeNow.Unix()
		}
	}
	if expireTime != 0 {
		return expireTime, delayTime
	}
	curTimeStr := timeNow.Format(FormatDate)
	targetTimeStr := fmt.Sprint("%s %s", curTimeStr, u.ExpireTimes)
	t2, _ := time.ParseInLocation(FormatDateTime, targetTimeStr, time.Local)
	//每天
	if u.Opt == nil || len(u.Opt) == 0 {
		if timeNow.Unix() >= t2.Unix() {
			expireTime = t2.AddDate(0, 0, 1).Unix()
		} else {
			expireTime = t2.Unix()
		}
	} else {
		//每周
		u.ExpireTimes = make([]int64, 0, 7)
		for _, weekday := range u.Opt {
			curWeekDay := timeNow.Weekday()
			internal := (7 + weekday - curWeekDay) % 7
			targetUnix := timeNow.Unix() + int64(internal)*secondsPerDay
			u.ExpireTimes = append(u.ExpireTimes, targetUnix)
			sort.Slice(u.ExpireTimes, func(i, j int) bool {
				return u.ExpireTimes[i] < u.ExpireTimes[i]
			})
		}
		return u.Refresh()
	}
	return expireTime, delayTime
}

type NodeList struct {
	//TimerNodeList []*TimeNode //后续可能改成链表结构
	root      *TimeNode
	timeWheel *TimeWheel
	lock      sync.RWMutex
}

func (t *NodeList) putNode(node *TimeNode) {

	if t.root != nil {
		t.root.prev = node
	}
	node.next = t.root
	node.prev = nil
	t.root = node
	t.timeWheel.nodeCount++
}

func (t *NodeList) removeNode(node *TimeNode) bool {
	if node.prev == nil && node.next == nil {
		return false
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node.prev != nil {

	}
	if node.prev != nil {
		// if not header
		node.prev.next = node.next
	} else {
		t.root = node.next
	}
	node.next = nil
	node.prev = nil
	t.timeWheel.nodeCount--
	if t.timeWheel.nodeCount == 0 {
		return true
	}
	return false
}

func (t *NodeList) foreachNode(handler func(node *TimeNode)) {
	for ch := t.root; ch != nil; ch = ch.next {
		handler(ch)
	}
}

func (t *NodeList) flush() {
	for node := t.root; node != nil; node = node.next {
		t.removeNode(node)
	}
}
