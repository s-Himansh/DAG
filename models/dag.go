package models

import "sync"

type NodeState int

const (
	StatePending NodeState = iota
	StateReady
	StateRunning
	StateCompleted
	StateFailed
	StateSkipped
)

func (s NodeState) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

type Node struct {
	ID           string
	Task         *Task
	Dependencies []string
	State        NodeState
	mu           sync.Mutex
}

func NewNode(id string, task *Task) *Node {
	return &Node{
		ID:           id,
		Task:         task,
		Dependencies: make([]string, 0),
		State:        StatePending,
	}
}

func (n *Node) SetState(state NodeState) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.State = state
}

func (n *Node) GetState() NodeState {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.State
}

type Edge struct {
	From string
	To   string
}

type DAG struct {
	Nodes map[string]*Node
	Edges []Edge
	mu    sync.RWMutex
}

func NewDAG() *DAG {
	return &DAG{
		Nodes: make(map[string]*Node),
		Edges: make([]Edge, 0),
	}
}

func (d *DAG) AddNode(node *Node) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Nodes[node.ID] = node
}

func (d *DAG) AddEdge(from, to string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.Nodes[from]; !ok {
		return
	}
	if _, ok := d.Nodes[to]; !ok {
		return
	}

	d.Edges = append(d.Edges, Edge{From: from, To: to})
	d.Nodes[to].Dependencies = append(d.Nodes[to].Dependencies, from)
}

func (d *DAG) GetNode(id string) (*Node, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	node, ok := d.Nodes[id]
	return node, ok
}

func (d *DAG) GetChildren(id string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	children := make([]string, 0)
	for _, edge := range d.Edges {
		if edge.From == id {
			children = append(children, edge.To)
		}
	}
	return children
}

func (d *DAG) GetEdges() []Edge {
	d.mu.RLock()
	defer d.mu.RUnlock()
	edges := make([]Edge, len(d.Edges))
	copy(edges, d.Edges)
	return edges
}

func (d *DAG) GetNodes() map[string]*Node {
	d.mu.RLock()
	defer d.mu.RUnlock()
	nodes := make(map[string]*Node, len(d.Nodes))
	for k, v := range d.Nodes {
		nodes[k] = v
	}
	return nodes
}

func (d *DAG) NodeCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.Nodes)
}

func (d *DAG) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, node := range d.Nodes {
		node.SetState(StatePending)
	}
}
