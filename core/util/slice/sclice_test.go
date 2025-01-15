package slice

import "testing"

func TestSlice(t *testing.T) {
	t.Log("Contains:", Contains(4, []int{1, 2, 3, 4, 5}))
	t.Log("RemoveDuplicate:", RemoveDuplicate([]int{1, 2, 3, 4, 5, 5, 4, 3, 2, 1}))
	t.Log("MergeRemoveDuplicate:", MergeRemoveDuplicate([]int{1, 2, 3}, []int{5, 4, 3, 2, 1, 6}))
}
