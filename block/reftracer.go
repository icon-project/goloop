package block

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common/log"
)

type RefCounter interface {
	RefCount() int
}

type RefTracer struct {
	Logger      log.Logger
	refCounters []RefCounter
}

func stringfy(rc RefCounter) string {
	if s, ok := rc.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%p(%d)", rc, rc.RefCount())
}

func (rt *RefTracer) objectsString() string {
	var buf bytes.Buffer
	for i, rc := range rt.refCounters {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(stringfy(rc))
	}
	return buf.String()
}

func (rt *RefTracer) TraceNew(rc RefCounter) {
	rt.refCounters = append(rt.refCounters, rc)
	rt.Logger.Debugf("new Obj:%s nObjs:%d Objs:%s\n", rc, len(rt.refCounters), rt.objectsString())
	// rt.Logger.Debugf("stack:%+v\n", errors.New("stack"))
}

func (rt *RefTracer) TraceRef(rc RefCounter) {
	rt.Logger.Debugf("ref Obj:%s\n", stringfy(rc))
}

func (rt *RefTracer) TraceUnref(rc RefCounter) {
	if rc.RefCount() != 0 {
		rt.Logger.Debugf("unref Obj:%s\n", stringfy(rc))
	} else {
		rt.traceDispose(rc)
		rt.Logger.Debugf("unref Obj:%s nObj:%d Objs:%s\n", stringfy(rc), len(rt.refCounters), rt.objectsString())
	}
}

func (rt *RefTracer) traceDispose(rc RefCounter) {
	for i, rci := range rt.refCounters {
		if rci == rc {
			last := len(rt.refCounters) - 1
			rt.refCounters[i] = rt.refCounters[last]
			rt.refCounters[last] = nil
			rt.refCounters = rt.refCounters[:last]
		}
	}
}
func (rt *RefTracer) TraceDispose(rc RefCounter) {
	rt.traceDispose(rc)
	rt.Logger.Debugf("dispose Obj:%s nObj:%d Objs:%s\n", stringfy(rc), len(rt.refCounters), rt.objectsString())
}
