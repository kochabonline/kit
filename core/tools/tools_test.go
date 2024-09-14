package tools

import (
	"testing"
)

func TestTools(t *testing.T) {
	t.Log("Id:", Id())
	t.Log("IpV4:", IpV4())
	t.Log("Hostname:", Hostname())
	t.Log("GenerateRandomCode:", GenerateRandomCode(6))
}
