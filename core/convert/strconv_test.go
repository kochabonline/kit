package convert

import "testing"

func TestStrconv(t *testing.T) {
	result, err := Strconv[int]("1", "2", "3")
	if err != nil {
		t.Error(err)
	}
	t.Log("Strconv:", result)
}
