package common

import (
	"sync"
	"time"
)

type Timer interface {
	Stop() bool
}

type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) Timer
}

type GoTimeClock struct {
}

func (cl *GoTimeClock) Now() time.Time {
	return time.Now()
}

func (cl *GoTimeClock) AfterFunc(d time.Duration, f func()) Timer {
	return time.AfterFunc(d, f)
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
	return aft
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
