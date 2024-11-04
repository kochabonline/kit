package convert

type Node interface {
	GetId() int64
	GetParentId() int64
	GetNode() Node
	SetChildren(children []Node)
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
