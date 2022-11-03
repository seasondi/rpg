package timerWheel

import (
	"time"
)

var (
	currentMaxTimerId = int64(0)
)

func TimeToMs(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func MsToTime(t int64) time.Time {
	return time.Unix(0, t*int64(time.Millisecond)).Local()
}

func nextTimerID() int64 {
	currentMaxTimerId += 1
	return currentMaxTimerId
}

func timeToPrecision(t int64, precision int64) int64 {
	return t / precision * precision
}

func truncate(startMs int64, expiration int64, stepMs int64) int64 {
	if expiration <= startMs {
		return startMs
	}
	diff := (expiration - startMs) / stepMs

	return startMs + diff*stepMs
}
