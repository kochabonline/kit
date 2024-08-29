package reflect

import "testing"

type Mock struct {
	Name    string   `default:"John"`
	Age     int      `default:"18"`
	Hobby   []string `default:"basketball,football"`
	Enabled bool     `default:"true"`
	Address struct {
		Province string `default:"New York"`
		City     string `default:"New York"`
	}
}

func TestTag(t *testing.T) {
	mock := Mock{Age: 20}

	if err := SetDefaultTag(&mock); err != nil {
		t.Fatal(err)
	}

	t.Log(mock)
}
