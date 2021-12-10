package time_wheel

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TimeWheel struct {
	wheelSize            int64
	tickMs               int64
	slot                 int64
	lock                 sync.RWMutex
	Nodes                map[uint64]*TimeNode
	currentTime          int64
	round                uint32
	bucket               []*NodeList
	nextWheel            *TimeWheel
	prevWheel            *TimeWheel
	receivePrevWheelChan chan *NodeList
	nodeCount            int32
}

func NewTimeWheel(slot int64, tickMs int64, wheelSize int64, startMs int64, round uint32, signalChan chan *NodeList) *TimeWheel {
	timeWheel := &TimeWheel{
		slot:                 slot,
		tickMs:               tickMs,
		wheelSize:            wheelSize,
		bucket:               make([]*NodeList, wheelSize),
		currentTime:          startMs,
		round:                round,
		receivePrevWheelChan: signalChan,
	}
	return timeWheel
}

func (t *TimeWheel) Start(ticker *time.Ticker, cmdChan chan cmd, receiveChan chan *TimeNode, quitChan chan struct{}) {

	for {
		select {
		case <-ticker.C:
			t.advanceClock(t.currentTime + t.tickMs)
		case nodeList, ok := <-t.receivePrevWheelChan:
			if !ok {
				break
			}
			//降级逻辑
			nodeList.foreachNode(func(node *TimeNode) {
				t.addTimerNode(node)
			})

		case node, ok := <-receiveChan:
			if !ok {
				break
			}
			t.addTimerNode(node)
		case cmd, ok := <-cmdChan:
			if !ok {
				return
			}
			cmd.run(t)
		case <-quitChan:
			t.stop()
			return
		}
	}
}

func (t *TimeWheel) loadNode() {

}

func (t *TimeWheel) addTimerNode(node *TimeNode) {
	//这个节点过期了
	if node.expireTime <= t.currentTime {
		node.signalChan <- struct{}{}
		return
	}
	remainTme := node.expireTime - t.currentTime
	//判断
	if remainTme < t.tickMs*t.wheelSize {
		//封装节点
		slot := (remainTme/t.tickMs + t.slot) % t.wheelSize
		nodeList := t.bucket[slot]
		if nodeList == nil {
			nodeList = &NodeList{timeWheel: t}
			t.bucket[slot] = nodeList
		}
		//添加到bucket中
		nodeList.putNode(node)
		//nodeList.TimerNodeList = append(nodeList.TimerNodeList, node)
		t.Nodes[node.NodeId] = node
		//t.NodeId2BucketMap[node.NodeId] = slot
	} else {
		if t.nextWheel == nil {
			t.nextWheel = NewTimeWheel(0, t.tickMs*t.wheelSize, t.wheelSize, t.currentTime, t.round+1, t.receivePrevWheelChan)
			t.nextWheel.receivePrevWheelChan = t.receivePrevWheelChan
		}
		t.nextWheel.addTimerNode(node)
	}
}

func (t *TimeWheel) removeNode(nodeId uint64) {
	if t.Nodes == nil {
		return
	}

	var (
		ok       bool
		nodeList *NodeList
		node     *TimeNode
	)
	if node, ok = t.Nodes[nodeId]; ok {
		nodeList = node.nodeList
		if node.NodeId == nodeId {
			delete(t.Nodes, nodeId)
		}
		if nodeList != nil && nodeList.removeNode(node) {
			t.removeTimeWheel()
		}
	} else if t.nextWheel != nil {
		t.nextWheel.removeNode(nodeId)
	}

}
func (t *TimeWheel) removeTimeWheel() {
	if t.round == 1 {
		return
	}
	if t.nextWheel == nil && t.prevWheel == nil {
		return
	}

	if t.nextWheel != nil {
		t.prevWheel.nextWheel = t.nextWheel
	}
	t.prevWheel.nextWheel = t.nextWheel
	t.prevWheel = nil
	t.nextWheel = nil
	t.stop()
}

func (t *TimeWheel) advanceClock(timeMs int64) {
	if timeMs < t.currentTime+t.tickMs {
		return
	}
	t.currentTime += t.tickMs
	if t.round == 0 {
		t.signalCaller()
	} else {
		t.signalLowerWheel()
	}
	t.slot = (t.slot + 1) % t.wheelSize

	if t.nextWheel == nil {
		return
	}
	t.nextWheel.advanceClock(timeMs)
}

func (t *TimeWheel) signalLowerWheel() {

	nodeList := t.bucket[t.slot]
	if nodeList == nil || nodeList.root == nil {
		return
	}
	rootNode := nodeList.root

	hand := func(node *TimeNode) {
		delete(t.Nodes, node.NodeId)
	}
	nodeList.foreachNode(hand)
	//nodeList.flush()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	select {
	case t.receivePrevWheelChan <- &NodeList{root: rootNode}:
		t.bucket[t.slot] = &NodeList{timeWheel: t, root: nil}
	case <-ctx.Done():
		fmt.Println("channel timeout")
	}

}

func (t *TimeWheel) signalCaller() {
	nodeList := t.bucket[t.slot]
	if nodeList == nil || nodeList.root == nil {
		return
	}
	t.bucket[t.slot] = &NodeList{timeWheel: t, root: nil}

	hand := func(node *TimeNode) {
		node.signalChan <- struct{}{}
		delete(t.Nodes, node.NodeId)
		//node如果是tick类型，需要重复加入队列中
		if node.timerType == TickTimerNode {
			node.expireTime, node.expireTime = node.refreshHandler.Refresh()
			t.addTimerNode(node)
		}

	}
	nodeList.foreachNode(hand)

	//nodeList.r = make([]*TimeNode, 0, t.wheelSize)
}

func (t *TimeWheel) stop() {

}
