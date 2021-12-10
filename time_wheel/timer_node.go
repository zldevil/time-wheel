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

//对象池？
type TimeNode struct {
	delayTime      int64
	expireTime     int64
	signalChan     chan struct{}
	NodeId         uint64
	timerType      TimerType
	refreshHandler NodeHandler
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

type internalCycle struct {
	delayTime int64
}

func (i *internalCycle) Refresh() (int64, int64) {
	expireTime := time.Now().Unix() + i.delayTime
	return expireTime, i.delayTime
}

type unixCycle struct {
	expireTimes []int64
	opt         []time.Weekday
	expireStr   string
}

func (u *unixCycle) Refresh() (int64, int64) {
	var (
		timeNow    = time.Now()
		delayTime  int64
		expireTime int64
	)

	for i := 0; i < len(u.expireTimes); i++ {
		if timeNow.Unix() < u.expireTimes[i] {
			expireTime = u.expireTimes[i]
			delayTime = expireTime - timeNow.Unix()
		}
	}
	if expireTime != 0 {
		return expireTime, delayTime
	}
	curTimeStr := timeNow.Format(FormatDate)
	targetTimeStr := fmt.Sprint("%s %s", curTimeStr, u.expireTimes)
	t2, _ := time.ParseInLocation(FormatDateTime, targetTimeStr, time.Local)
	//每天
	if u.opt == nil || len(u.opt) == 0 {
		if timeNow.Unix() >= t2.Unix() {
			expireTime = t2.AddDate(0, 0, 1).Unix()
		} else {
			expireTime = t2.Unix()
		}
	} else {
		//每周
		u.expireTimes = make([]int64, 0, 7)
		for _, weekday := range u.opt {
			curWeekDay := timeNow.Weekday()
			internal := (7 + weekday - curWeekDay) % 7
			targetUnix := timeNow.Unix() + int64(internal)*secondsPerDay
			u.expireTimes = append(u.expireTimes, targetUnix)
			sort.Slice(u.expireTimes, func(i, j int) bool {
				return u.expireTimes[i] < u.expireTimes[i]
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
