package reflect

import "testing"

type mapMock struct {
	Name    string `json:"name"`
	Age     int
	Address struct {
		Province string `json:"province"`
		City     string `json:"City"`
	}
}

func TestMap(t *testing.T) {
	mock := mapMock{Name: "John", Age: 0, Address: struct {
		Province string `json:"province"`
		City     string `json:"City"`
	}{Province: "Jawa Barat", City: "Bandung"},
	}

	result, err := StructConvMap(mock)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)
}
