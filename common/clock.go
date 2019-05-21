package common

import (
	"sync"
	"time"
)

type Timer struct {
	ITimer
	C <-chan time.Time
}

func newTimer(itm ITimer) Timer {
	return Timer{itm, make(chan time.Time)}
}

func newTimerWithChan(itm ITimer, c <-chan time.Time) Timer {
	return Timer{itm, c}
}

func (timer *Timer) Stop() {
	timer.ITimer.Stop()
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
	return newTimer(time.AfterFunc(d, f))
}

func (cl *GoTimeClock) NewTimer(d time.Duration) Timer {
	return newTimer(time.NewTimer(d))
}

func (cl *GoTimeClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

type afterFuncTimer struct {
	cl *TestClock
	t  time.Time
	f  func()
}

func (timer *afterFuncTimer) Stop() bool {
	timer.cl.Lock()
	defer timer.cl.Unlock()

	if timer.f == nil {
		return false
	}
	timer.f = nil

	for i, tm := range timer.cl.afterFuncTimers {
		if timer == tm {
			last := len(timer.cl.afterFuncTimers) - 1
			timer.cl.afterFuncTimers[i] = timer.cl.afterFuncTimers[last]
			timer.cl.afterFuncTimers[last] = nil
			timer.cl.afterFuncTimers = timer.cl.afterFuncTimers[:last]
		}
	}
	return true
}

type TestClock struct {
	sync.Mutex
	now             time.Time
	afterFuncTimers []*afterFuncTimer
}

func (cl *TestClock) Now() time.Time {
	cl.Lock()
	defer cl.Unlock()

	return cl.now
}

func (cl *TestClock) AfterFunc(d time.Duration, f func()) Timer {
	cl.Lock()
	defer cl.Unlock()

	t := cl.now.Add(d)
	aft := &afterFuncTimer{cl, t, f}
	cl.afterFuncTimers = append(cl.afterFuncTimers, aft)
	return newTimer(aft)
}

func (cl *TestClock) NewTimer(d time.Duration) Timer {
	cl.Lock()
	defer cl.Unlock()

	t := cl.now.Add(d)
	c := make(chan time.Time)
	aft := &afterFuncTimer{cl, t, func() {
		c <- cl.Now()
	}}
	cl.afterFuncTimers = append(cl.afterFuncTimers, aft)
	return newTimerWithChan(aft, c)
}

func (cl *TestClock) Sleep(d time.Duration) {
	timer := cl.NewTimer(d)
	<-timer.C
}

func (cl *TestClock) PassTime(d time.Duration) {
	cl.SetTime(cl.now.Add(d))
}

func (cl *TestClock) SetTime(t time.Time) {
	var timers []*afterFuncTimer

	cl.Lock()
	defer func() {
		for _, tm := range timers {
			tm.f()
		}
	}()
	defer cl.Unlock()

	if t.Before(cl.now) {
		return
	}
	cl.now = t
	for i := 0; i < len(cl.afterFuncTimers); {
		tm := cl.afterFuncTimers[i]
		if cl.now.Equal(tm.t) || cl.now.After(tm.t) {
			last := len(cl.afterFuncTimers) - 1
			cl.afterFuncTimers[i] = cl.afterFuncTimers[last]
			cl.afterFuncTimers[last] = nil
			cl.afterFuncTimers = cl.afterFuncTimers[:last]
			timers = append(timers, tm)
			continue
		}
		i++
	}
}

func UnixMicroFromTime(t time.Time) int64 {
	return t.UnixNano() / int64(time.Microsecond)
}
