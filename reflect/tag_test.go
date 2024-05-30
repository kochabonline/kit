package reflect

import "testing"

func TestTag(t *testing.T) {
	type Mock struct {
		Name string `default:"John"`
		Age  int    `default:"18"`
		Hobby struct {
			Basketball string `default:"basketball"`
			Football string `default:"football"`
		}
	}

	mock := Mock{ Age: 20 }

	if err := SetDefaultTag(&mock); err != nil {
		t.Fatal(err)
	}

	t.Log(mock)
}
