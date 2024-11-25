package scheduler

type Gossip struct {
	nodes map[string]*Node
}

func NewGossip() *Gossip {
	return &Gossip{
		nodes: make(map[string]*Node),
	}
}

func (g *Gossip) AddNode(node *Node) {
	g.nodes[node.Id] = node

	for _, n := range g.nodes {
		n.hashRing.Add(node.Id)
		node.hashRing.Add(n.Id)
	}
}

func (g *Gossip) RemoveNode(node *Node) {
	delete(g.nodes, node.Id)

	for _, n := range g.nodes {
		n.hashRing.Remove(node.Id)
		n.HandleFailure(node.Id)
	}
}
