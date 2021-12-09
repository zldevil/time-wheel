package time_wheel

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TimeWheel struct {
	wheelSize                int64
	tickMs                   int64
	slot                     int64
	lock                     sync.RWMutex
	NodeId2BucketMap         map[uint64]int64
	currentTime              int64
	round                    uint32
	bucket                   []*TimeNodeList
	overFlowWheel            *TimeWheel
	receiveOverFlowWheelChan chan *TimeNodeList
}

func NewTimeWheel(slot int64, tickMs int64, wheelSize int64, startMs int64, round uint32, signalChan chan *TimeNodeList) *TimeWheel {
	timeWheel := &TimeWheel{
		slot:                     slot,
		tickMs:                   tickMs,
		wheelSize:                wheelSize,
		bucket:                   make([]*TimeNodeList, wheelSize),
		currentTime:              startMs,
		round:                    round,
		receiveOverFlowWheelChan: signalChan,
	}
	return timeWheel
}

func (t *TimeWheel) Start(ticker *time.Ticker, cmdChan chan cmd, receiveChan chan *TimeNode, quitChan chan struct{}) {

	for {
		select {
		case <-ticker.C:
			t.advanceClock(t.currentTime + t.tickMs)
		case timeNodeList, ok := <-t.receiveOverFlowWheelChan:
			if !ok {
				break
			}
			//降级逻辑
			for _, node := range timeNodeList.TimerNodeList {
				t.addTimerNode(node)
			}
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

func (t *TimeWheel) stop() {

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
			nodeList = &TimeNodeList{TimerNodeList: make([]*TimeNode, 0, t.wheelSize)}
			t.bucket[slot] = nodeList
		}
		//添加到bucket中
		nodeList.TimerNodeList = append(nodeList.TimerNodeList, node)
		t.NodeId2BucketMap[node.NodeId] = slot
	} else {
		if t.overFlowWheel == nil {
			t.overFlowWheel = NewTimeWheel(0, t.tickMs*t.wheelSize, t.wheelSize, t.currentTime, t.round+1, t.receiveOverFlowWheelChan)
			t.overFlowWheel.receiveOverFlowWheelChan = t.receiveOverFlowWheelChan
		}
		t.overFlowWheel.addTimerNode(node)
	}
}

func (t *TimeWheel) removeNode(nodeId uint64) {
	if t.NodeId2BucketMap == nil {
		return
	}
	slot, ok := t.NodeId2BucketMap[nodeId]
	if ok {
		//删除bucket中对应的节点
		nodeList := t.bucket[slot].TimerNodeList
		for i := 0; i < len(nodeList); i++ {
			if nodeList[i].NodeId != nodeId {
				continue
			}
			nodeList = append(nodeList[0:i], nodeList[i+1:]...)
		}
		delete(t.NodeId2BucketMap, nodeId)
		return
	}
	if t.overFlowWheel != nil {
		t.overFlowWheel.removeNode(nodeId)
	}

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

	if t.overFlowWheel == nil {
		return
	}
	t.overFlowWheel.advanceClock(timeMs)
}

func (t *TimeWheel) signalLowerWheel() {

	nodeList := t.bucket[t.slot]
	if nodeList == nil || len(nodeList.TimerNodeList) == 0 {
		return
	}

	for _, node := range nodeList.TimerNodeList {
		delete(t.NodeId2BucketMap, node.NodeId)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	select {
	case t.receiveOverFlowWheelChan <- nodeList:
	case <-ctx.Done():
		fmt.Println("channel timeout")
	}

	t.bucket[t.slot] = &TimeNodeList{TimerNodeList: make([]*TimeNode, 0, t.wheelSize)}
}

func (t *TimeWheel) signalCaller() {
	nodeList := t.bucket[t.slot]
	if nodeList == nil || len(nodeList.TimerNodeList) == 0 {
		return
	}

	for _, node := range nodeList.TimerNodeList {
		node.signalChan <- struct{}{}
		//node如果是tick类型，需要重复加入队列中
		if node.timerType == DelayTimerNode {
			continue
		}
		node.expireTime, node.expireTime = node.refreshHandler.Refresh()
		t.addTimerNode(node)
		//close(node.signalChan)
	}
	nodeList.TimerNodeList = make([]*TimeNode, 0, t.wheelSize)
}

func (t *TimeWheel) afterTime(duration int64) (chan struct{}, error) {

	//判断时间是否超出当前时间轮的最大时间
	/*	t.lock.Lock()
		defer t.lock.Unlock()*/

	if duration < t.tickMs*t.wheelSize {
		//封装节点
		node := &TimeNode{delayTime: duration, signalChan: make(chan struct{}, 1), timerType: 1}
		slot := (duration/t.tickMs + t.slot) % t.wheelSize
		nodeList := t.bucket[slot]
		if nodeList == nil {
			nodeList = &TimeNodeList{TimerNodeList: make([]*TimeNode, 0, t.wheelSize)}
			t.bucket[slot-1] = nodeList
		}
		//添加到bucket中
		nodeList.TimerNodeList = append(nodeList.TimerNodeList, node)
		//计算slot
		return node.signalChan, nil
	} else {
		if t.overFlowWheel == nil {
			t.overFlowWheel = NewTimeWheel(0, t.tickMs*t.wheelSize, t.wheelSize, t.currentTime, t.round+1, t.receiveOverFlowWheelChan)
		}
		return t.overFlowWheel.afterTime(duration)
	}
}
