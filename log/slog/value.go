package slog

import (
	"runtime"
	"strconv"
	"strings"
)

func caller(depth int) string {
	_, file, line, _ := runtime.Caller(depth)
	idx := strings.LastIndexByte(file, '/')
	if idx == -1 {
		return file[idx+1:] + ":" + strconv.Itoa(line)
	}
	idx = strings.LastIndexByte(file[:idx], '/')
	return file[idx+1:] + ":" + strconv.Itoa(line)
}

func callerSkipFrameCount() int {
	// Ask runtime.Callers for up to 10 pcs, including runtime.Callers itself.
	pc := make([]uintptr, 10)
	n := runtime.Callers(3, pc)
	if n == 0 {
		return 0
	}

	pc = pc[:n] // pass only valid pcs to runtime.CallersFrames
	frames := runtime.CallersFrames(pc)

	// Loop to get frames.
	// A fixed number of pcs can expand to an indefinite number of Frames.
	frameCount := 0
	for {
		_, more := frames.Next()
		if !more {
			break
		}
		frameCount++
	}

	return frameCount
}
