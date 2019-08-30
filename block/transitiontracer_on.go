// +build tracetr

package block

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/icon-project/goloop/common/log"
)

type tracer struct {
	sync.Mutex
	counter  int32
	liveTRIs []*transitionImpl
}

var trc tracer

func (trc *tracer) _liveTRIs() string {
	var buf bytes.Buffer
	for _, tri := range trc.liveTRIs {
		buf.WriteString(fmt.Sprintf(" %p(%d) ", tri, tri._nRef))
	}
	return buf.String()
}

func traceNewTransitionImpl(tri *transitionImpl) {
	trc.Lock()
	defer trc.Unlock()

	trc.counter++
	trc.liveTRIs = append(trc.liveTRIs, tri)
	log.Debugf("new TRI=%p nRef=%d counter=%d live:%s\n", tri, tri._nRef, trc.counter, trc._liveTRIs())
	// log.Debugf("stack:%+v\n", errors.New("stack"))
}

func traceRef(tri *transitionImpl) {
	log.Debugf("ref tri=%p nRef=%d\n", tri, tri._nRef)
	// log.Debugf("stack:%+v\n", errors.New("stack"))
}

func traceUnref(tri *transitionImpl) {
	trc.Lock()
	defer trc.Unlock()

	if tri._nRef != 0 {
		log.Debugf("unref TRI=%p nRef=%d\n", tri, tri._nRef)
	} else {
		trc.counter--
		for i, ltri := range trc.liveTRIs {
			if ltri == tri {
				last := len(trc.liveTRIs) - 1
				trc.liveTRIs[i] = trc.liveTRIs[last]
				trc.liveTRIs[last] = nil
				trc.liveTRIs = trc.liveTRIs[:last]
			}
		}
		log.Debugf("unref TRI=%p nRef=%d counter=%d live:%s\n", tri, tri._nRef, trc.counter, trc._liveTRIs())
	}
	// log.Debugf("stack:%+v\n", errors.New("stack"))
}
