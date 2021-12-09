package time_wheel

type cmd interface {
	run(t *TimeWheel)
}

type addNodeCmd struct {
	node *TimeNode
}

func (a *addNodeCmd) run(t *TimeWheel) {
	t.addTimerNode(a.node)
}

type removeNodeCmd struct {
	nodeId uint64
}

func (r *removeNodeCmd) run(t *TimeWheel) {
	t.removeNode(r.nodeId)
}
