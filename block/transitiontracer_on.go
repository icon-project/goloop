// +build tracetr

package block

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/icon-project/goloop/common/log"
)

type tracker struct {
	sync.Mutex
	counter  int32
	liveTRIs []*transitionImpl
}

var t tracker

func (t *tracker) _liveTRIs() string {
	var buf bytes.Buffer
	for _, tri := range t.liveTRIs {
		buf.WriteString(fmt.Sprintf(" %p(%d) ", tri, tri._nRef))
	}
	return buf.String()
}

func traceNewTransitionImpl(tri *transitionImpl) {
	t.Lock()
	defer t.Unlock()

	t.counter++
	t.liveTRIs = append(t.liveTRIs, tri)
	log.Debugf("new TRI=%p nRef=%d counter=%d live:%s\n", tri, tri._nRef, t.counter, t._liveTRIs())
	// log.Debugf("stack:%+v\n", errors.New("stack"))
}

func traceRef(tri *transitionImpl) {
	log.Debugf("ref tri=%p nRef=%d\n", tri, tri._nRef)
	// log.Debugf("stack:%+v\n", errors.New("stack"))
}

func traceUnref(tri *transitionImpl) {
	t.Lock()
	defer t.Unlock()

	if tri._nRef != 0 {
		log.Debugf("unref TRI=%p nRef=%d\n", tri, tri._nRef)
	} else {
		t.counter--
		for i, ltri := range t.liveTRIs {
			if ltri == tri {
				last := len(t.liveTRIs) - 1
				t.liveTRIs[i] = t.liveTRIs[last]
				t.liveTRIs[last] = nil
				t.liveTRIs = t.liveTRIs[:last]
			}
		}
		log.Debugf("unref TRI=%p nRef=%d counter=%d live:%s\n", tri, tri._nRef, t.counter, t._liveTRIs())
	}
	// log.Debugf("stack:%+v\n", errors.New("stack"))
}
