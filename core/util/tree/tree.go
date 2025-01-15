package tree

import "sort"

type Node interface {
	GetNode() Node
	SetChildren(children []Node)
	GetId() int64
	GetParentId() int64
}

type Sortable interface {
	GetOrder() int
}

func ConvertToNodeSlice[T Node](data []T) []Node {
	nodes := make([]Node, len(data))
	for i, d := range data {
		nodes[i] = d
	}
	return nodes
}

func buildNodeMap(nodes []Node) map[int64][]Node {
	nodeMap := make(map[int64][]Node, len(nodes))
	for _, node := range nodes {
		parentId := node.GetParentId()
		nodeMap[parentId] = append(nodeMap[parentId], node)
	}

	// Sort children by order if they are sortable
	for _, children := range nodeMap {
		sort.Slice(children, func(i, j int) bool {
			a, aOk := children[i].(Sortable)
			b, bOk := children[j].(Sortable)
			if !aOk || !bOk {
				return false
			}
			return a.GetOrder() < b.GetOrder()
		})
	}

	return nodeMap
}

func buildTree(nodeMap map[int64][]Node, parentId int64) []Node {
	children, exists := nodeMap[parentId]
	if !exists {
		return []Node{}
	}

	tree := make([]Node, len(children))
	for i, child := range children {
		childNode := child.GetNode()
		childId := child.GetId()
		childNode.SetChildren(buildTree(nodeMap, childId))
		tree[i] = childNode
	}
	return tree
}

func BuildTree(nodes []Node, parentId int64) []Node {
	nodeMap := buildNodeMap(nodes)
	return buildTree(nodeMap, parentId)
}
