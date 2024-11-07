package tools

import (
	"math/rand"
	"net"
	"os"
	"time"

	"strings"

	"github.com/google/uuid"
)

func Id() string {
	return uuid.New().String()
}

func IpV4() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return ""
	}
	defer conn.Close()

	addr := conn.LocalAddr().(*net.UDPAddr)
	ip := strings.Split(addr.String(), ":")[0]
	return ip
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}

	return hostname
}

// GenerateRandomCode generates a random code with the specified length.
func GenerateRandomCode(length int) string {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	digits := "0123456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = digits[r.Intn(len(digits))]
	}
	return string(code)
}

// Contains compares a value with a list of values and returns true if the value is found in the list.
func Contains[T comparable](a T, list []T) bool {
	const threshold = 50

	if len(list) < threshold {
		for _, b := range list {
			if b == a {
				return true
			}
		}
		return false
	}

	set := make(map[T]struct{}, len(list))
	for _, b := range list {
		set[b] = struct{}{}
	}
	_, found := set[a]
	return found
}

// RemoveDuplicate removes duplicate values from a list.
func RemoveDuplicate[T comparable](list []T) []T {
	set := make(map[T]struct{}, len(list))
	for _, item := range list {
		set[item] = struct{}{}
	}

	result := make([]T, 0, len(set))
	for item := range set {
		result = append(result, item)
	}
	return result
}
