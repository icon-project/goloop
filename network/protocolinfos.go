package network

import (
	"sort"
	"sync"

	"github.com/icon-project/goloop/module"
)

type ProtocolInfos struct {
	m   map[byte][]module.ProtocolInfo
	l   []module.ProtocolInfo
	mtx sync.RWMutex
}

func (pis *ProtocolInfos) indexOf(l []module.ProtocolInfo, pi module.ProtocolInfo) int {
	for i, v := range l {
		if pi.Uint16() == v.Uint16() {
			return i
		}
	}
	return -1
}

func (pis *ProtocolInfos) remove(l []module.ProtocolInfo, idx int) []module.ProtocolInfo {
	if idx < 0 || idx >= len(l) {
		return l
	}
	if idx == 0 {
		l = l[1:]
	} else if idx == (len(l) - 1) {
		l = l[:idx]
	} else {
		l = append(l[:idx], l[idx+1:]...)
	}
	return l
}

func (pis *ProtocolInfos) Add(pi module.ProtocolInfo) {
	pis.mtx.Lock()
	defer pis.mtx.Unlock()

	l, ok := pis.m[pi.ID()]
	if !ok {
		l = make([]module.ProtocolInfo, 1)
		l[0] = pi
	} else {
		if pis.indexOf(l, pi) >= 0 {
			return
		}
		l = append(l, pi)
		sort.Slice(l, func(i, j int) bool {
			return l[i].Version() > l[j].Version()
		})
	}
	pis.m[pi.ID()] = l
	pis.l = append(pis.l, pi)
}

func (pis *ProtocolInfos) Set(piList []module.ProtocolInfo) {
	pis.mtx.Lock()
	defer pis.mtx.Unlock()

	pis.m = make(map[byte][]module.ProtocolInfo)
	pis.l = make([]module.ProtocolInfo, 0)
	for _, pi := range piList {
		l, ok := pis.m[pi.ID()]
		if !ok {
			l = make([]module.ProtocolInfo, 1)
			l[0] = pi
		} else {
			if pis.indexOf(l, pi) >= 0 {
				continue
			}
			l = append(l, pi)
		}
		pis.m[pi.ID()] = l
		pis.l = append(pis.l, pi)
	}
	for id, l := range pis.m {
		if len(l) > 0 {
			sort.Slice(l, func(i, j int) bool {
				return l[i].Version() > l[j].Version()
			})
			pis.m[id] = l
		}
	}
}

func (pis *ProtocolInfos) Remove(pi module.ProtocolInfo) {
	pis.mtx.Lock()
	defer pis.mtx.Unlock()

	l, ok := pis.m[pi.ID()]
	if ok {
		if idx := pis.indexOf(l, pi); idx >= 0 {
			l = pis.remove(l, idx)
			if len(l) == 0 {
				delete(pis.m, pi.ID())
			} else {
				pis.m[pi.ID()] = l
			}
			idx = pis.indexOf(pis.l, pi)
			pis.l = pis.remove(pis.l, idx)
		}
	}
}

func (pis *ProtocolInfos) Exists(pi module.ProtocolInfo) bool {
	pis.mtx.RLock()
	defer pis.mtx.RUnlock()

	if l, ok := pis.m[pi.ID()]; ok {
		if pis.indexOf(l, pi) >= 0 {
			return true
		}
	}
	return false
}

func (pis *ProtocolInfos) ExistsByID(piList ...module.ProtocolInfo) bool {
	pis.mtx.RLock()
	defer pis.mtx.RUnlock()

	for _, pi := range piList {
		if l, ok := pis.m[pi.ID()]; !ok || len(l) == 0 {
			return false
		}
	}
	return true
}

func (pis *ProtocolInfos) Array() []module.ProtocolInfo {
	pis.mtx.RLock()
	defer pis.mtx.RUnlock()

	l := make([]module.ProtocolInfo, len(pis.l))
	copy(l, pis.l)
	return l
}

func (pis *ProtocolInfos) Resolve(target *ProtocolInfos) {
	pis.mtx.Lock()
	defer pis.mtx.Unlock()

	m := make(map[byte][]module.ProtocolInfo)
	l := make([]module.ProtocolInfo, 0)
	for id, tpil := range target.m {
		if pil, ok := pis.m[id]; ok {
		loop:
			for _, tpi := range tpil {
				if idx := pis.indexOf(pil, tpi); idx >= 0 {
					pil[0] = tpi
					m[id] = pil[:1]
					l = append(l, tpi)
					break loop
				}
			}
		}
	}
	pis.m = m
	pis.l = l
}

func (pis *ProtocolInfos) Len() int {
	return len(pis.l)
}

func (pis *ProtocolInfos) LenOfIDSet() int {
	return len(pis.m)
}

func newProtocolInfos() *ProtocolInfos {
	return &ProtocolInfos{
		m: make(map[byte][]module.ProtocolInfo),
		l: make([]module.ProtocolInfo, 0),
	}
}
