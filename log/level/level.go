package level

import (
	"fmt"
)

type Level int

const (
	Trace Level = -8
	Debug Level = -4
	Info  Level = 0
	Warn  Level = 4
	Error Level = 8
	Fatal Level = 12
)

func (l Level) Level() Level { return l }

func (l Level) String() string {
	str := func(base string, val Level) string {
		if val == 0 {
			return base
		}
		return fmt.Sprintf("%s%+d", base, val)
	}

	switch {
	case l < Debug:
		return str("TRACE", l-Trace)
	case l < Info:
		return str("DEBUG", l-Debug)
	case l < Warn:
		return str("INFO", l-Info)
	case l < Error:
		return str("WARN", l-Warn)
	case l < Fatal:
		return str("ERROR", l-Error)
	default:
		return str("FATAL", l-Fatal)
	}
}
