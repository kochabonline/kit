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

type Api struct {
	Name    string            `json:"name"`
	Method  string            `json:"method" default:"GET"`
	Url     string            `json:"url"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Timeout int               `json:"timeout" default:"3"`
	Period  int               `json:"period" default:"10"`
}

type mockWithSlice struct {
	Host    string  `json:"host"`
	Port    int     `json:"port" default:"8080"`
	Number  float64 `json:"number"`
	Enabled bool    `json:"enabled" default:"true"`
	Mock1   struct {
		Host string `json:"host" default:"localhost"`
	} `json:"mock1"`
	Mock2 struct {
		Number []int `json:"number"`
	} `json:"mock2"`
	Apis []Api `json:"apis"`
}

func TestTag(t *testing.T) {
	mock := tagMock{Age: 20}

	if err := SetDefaultTag(&mock); err != nil {
		t.Fatal(err)
	}

	t.Log(mock)
}

func TestTagWithSliceStruct(t *testing.T) {
	mock := &mockWithSlice{}

	// 初始化切片，但不设置默认值
	mock.Apis = []Api{
		{
			Name: "test-api",
			Url:  "http://example.com/api/test",
		},
	}

	t.Logf("Before setting defaults: %+v", mock)
	for i, api := range mock.Apis {
		t.Logf("  Api[%d]: %+v", i, api)
	}

	if err := SetDefaultTag(mock); err != nil {
		t.Fatal(err)
	}

	t.Logf("After setting defaults: %+v", mock)
	for i, api := range mock.Apis {
		t.Logf("  Api[%d]: %+v", i, api)
	}
}
