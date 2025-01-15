package tree

import (
	"encoding/json"
	"testing"
)

type TreeMock struct {
	Id       int64
	ParentId int64
	Title    string
	Order    int
	Children []*TreeMock
}

func (t *TreeMock) GetId() int64 {
	return t.Id
}

func (t *TreeMock) GetParentId() int64 {
	return t.ParentId
}

func (t *TreeMock) GetNode() Node {
	return t
}

func (t *TreeMock) GetOrder() int {
	return t.Order
}

func (t *TreeMock) SetChildren(children []Node) {
	t.Children = make([]*TreeMock, len(children))
	for i, c := range children {
		t.Children[i] = c.(*TreeMock)
	}
}

func TestBuildTree(t *testing.T) {
	data := []*TreeMock{
		{Id: 1, ParentId: 0, Title: "1", Order: 2},
		{Id: 2, ParentId: 1, Title: "1.1", Order: 4},
		{Id: 3, ParentId: 1, Title: "1.2", Order: 3},
		{Id: 4, ParentId: 0, Title: "2", Order: 1},
		{Id: 5, ParentId: 4, Title: "2.1"},
		{Id: 6, ParentId: 5, Title: "2.1.1"},
	}

	tree := BuildTree(ConvertToNodeSlice(data), 0)
	bytes, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		t.Error(err)
	}

	t.Log(string(bytes))
}
