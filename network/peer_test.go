package network

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_PeerRTT(t *testing.T) {
	sleepTime := 100 * time.Millisecond
	r := NewPeerRTT()
	r.Start()
	time.Sleep(sleepTime)
	actual := r.Stop()
	expected := r.et.Sub(r.st)
	assert.Equal(t, expected, actual)
	last, avg := r.Value()
	assert.Equal(t, expected, last)
	assert.Equal(t, expected, avg)
	converted := float64(expected) / float64(time.Millisecond)
	assert.Equal(t, converted, r.Last(time.Millisecond))
	assert.Equal(t, converted, r.Avg(time.Millisecond))
	t.Log(r.String())

	wg := sync.WaitGroup{}
	wg.Add(1)
	r.StartWithAfterFunc(sleepTime-time.Millisecond, func() {
		wg.Done()
	})
	time.Sleep(sleepTime)
	actual = r.Stop()
	timer := time.AfterFunc(time.Second, func() {
		assert.FailNow(t, "timeout")
	})
	wg.Wait()
	timer.Stop()
	assert.Equal(t, r.et.Sub(r.st), actual)
	//exponential weighted moving average model
	//avg = (1-0.125)*avg + 0.125*last
	fv := 0.875*float64(avg) + 0.125*float64(actual)
	_, avg = r.Value()
	assert.Equal(t, time.Duration(fv), avg)
	t.Log(r.String())
}

func Test_PeerRoleFlag(t *testing.T) {
	pr := p2pRoleNone
	assert.False(t, pr.Has(p2pRoleSeed))
	assert.False(t, pr.Has(p2pRoleRoot))
	assert.Equal(t, p2pRoleNone, pr)

	pr.SetFlag(p2pRoleSeed)
	assert.True(t, pr.Has(p2pRoleSeed))
	assert.False(t, pr.Has(p2pRoleRoot))
	assert.Equal(t, p2pRoleSeed, pr)

	pr.SetFlag(p2pRoleRoot)
	assert.True(t, pr.Has(p2pRoleSeed))
	assert.True(t, pr.Has(p2pRoleRoot))

	assert.Equal(t, p2pRoleSeed|p2pRoleRoot, pr)
}
