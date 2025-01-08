package reflect

import "testing"

type tagMock struct {
	Name    string    `default:"John"`
	Age     int       `default:"18"`
	Hobby   []string  `default:"basketball,football"`
	Score   []int     `default:"90,80"`
	Height  []float64 `default:"180.5,170.5"`
	Bmi     []bmi     `default:"{180.5 70.5},{170.5 60.5}"`
	Enabled bool      `default:"true"`
	Address struct {
		Province string `default:"New York"`
		City     string `default:"New York"`
	}
}

type bmi struct {
	Height float64 `default:"180.5"`
	Weight float64 `default:"70.5"`
}

func TestTag(t *testing.T) {
	mock := tagMock{Age: 20}

	if err := SetDefaultTag(&mock); err != nil {
		t.Fatal(err)
	}

	t.Log(mock)
}
