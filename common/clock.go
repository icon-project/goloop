package common

import (
	"time"
)

type Timer struct {
	itimer ITimer
	C      <-chan time.Time
}

func NewTimer(itm ITimer) Timer {
	return Timer{itm, make(chan time.Time)}
}

func NewTimerWithChan(itm ITimer, c chan time.Time) Timer {
	return Timer{itm, c}
}

func (timer *Timer) Stop() {
	timer.itimer.Stop()
}

type ITimer interface {
	Stop() bool
}

type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) Timer
	NewTimer(d time.Duration) Timer
	Sleep(d time.Duration)
}

type GoTimeClock struct {
}

func (cl *GoTimeClock) Now() time.Time {
	return time.Now()
}

func (cl *GoTimeClock) AfterFunc(d time.Duration, f func()) Timer {
	return NewTimer(time.AfterFunc(d, f))
}

func (cl *GoTimeClock) NewTimer(d time.Duration) Timer {
	return NewTimer(time.NewTimer(d))
}

func (cl *GoTimeClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func UnixMicroFromTime(t time.Time) int64 {
	return t.UnixNano() / int64(time.Microsecond)
}
