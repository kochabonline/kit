package zerolog

import (
	"runtime"
)

func callerSkipFrameCount(count int) int {
	// Ask runtime.Callers for up to 10 pcs, including runtime.Callers itself.
	pc := make([]uintptr, 10)
	n := runtime.Callers(count, pc)
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
