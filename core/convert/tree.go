package convert

type Node interface {
	GetId() int64
	GetParentId() int64
	GetNode() Node
	SetChildren(children []Node)
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

	tree := make([]Node, 0, len(children))
	for _, child := range children {
		childNode := child.GetNode()
		childId := child.GetId()
		childNode.SetChildren(buildTree(nodeMap, childId))
		tree = append(tree, childNode)
	}
	return tree
}

func BuildTree(nodes []Node, parentId int64) []Node {
	nodeMap := buildNodeMap(nodes)
	return buildTree(nodeMap, parentId)
}
