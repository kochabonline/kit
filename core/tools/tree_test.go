package tools

import (
	"encoding/json"
	"testing"
)

type TreeMock struct {
	Id       int
	ParentId int
	Title    string
	Children []*TreeMock
}

func (t *TreeMock) GetId() int {
	return t.Id
}

func (t *TreeMock) GetParentId() int {
	return t.ParentId
}

func (t *TreeMock) GetNode() Node {
	return t
}

func (t *TreeMock) SetChildren(children []Node) {
	t.Children = make([]*TreeMock, len(children))
	for i, c := range children {
		t.Children[i] = c.(*TreeMock)
	}
}

func TestBuildTree(t *testing.T) {
	data := []*TreeMock{
		{Id: 1, ParentId: 0, Title: "1"},
		{Id: 2, ParentId: 1, Title: "1.1"},
		{Id: 3, ParentId: 1, Title: "1.2"},
		{Id: 4, ParentId: 0, Title: "2"},
		{Id: 5, ParentId: 4, Title: "2.1"},
		{Id: 6, ParentId: 5, Title: "2.1.1"},
	}

	nodes := make([]Node, len(data))
	for i, d := range data {
		nodes[i] = d
	}
	tree := BuildTree(nodes, 0)
	bytes, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		t.Error(err)
	}

	t.Log(string(bytes))
}
