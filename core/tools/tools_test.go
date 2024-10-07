package tools

import (
	"testing"
)

func TestTools(t *testing.T) {
	t.Log("Id:", Id())
	t.Log("IpV4:", IpV4())
	t.Log("Hostname:", Hostname())
	t.Log("GenerateRandomCode:", GenerateRandomCode(6))
	t.Log("Contains:", Contains(4, []int{1, 2, 3, 4, 5}))
}
