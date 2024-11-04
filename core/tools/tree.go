package tools

type Node interface {
	GetId() int
	GetParentId() int
	GetNode() Node
	SetChildren(children []Node)
}

func buildNodeMap(nodes []Node) map[int][]Node {
	nodeMap := make(map[int][]Node, len(nodes))
	for _, node := range nodes {
		parentId := node.GetParentId()
		nodeMap[parentId] = append(nodeMap[parentId], node)
	}
	return nodeMap
}

func buildTree(nodeMap map[int][]Node, parentId int) []Node {
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

func BuildTree(nodes []Node, parentId int) []Node {
	nodeMap := buildNodeMap(nodes)
	return buildTree(nodeMap, parentId)
}
