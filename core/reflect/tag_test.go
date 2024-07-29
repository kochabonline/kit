package reflect

import "testing"

type Mock struct {
	Name  string `default:"John"`
	Age   int    `default:"18"`
	Hobby struct {
		Basketball string `default:"basketball"`
		Football   string `default:"football"`
	}
	Enabled bool `json:"enabled" default:"true"`
}

func TestTag(t *testing.T) {
	mock := Mock{Age: 20}

	if err := SetDefaultTag(&mock); err != nil {
		t.Fatal(err)
	}

	t.Log(mock)
}
