package eeproxy

import (
	"github.com/icon-project/goloop/common/ipc"
	"github.com/pkg/errors"
	"sync"
)

type scoreType int

const (
	pythonSCORE scoreType = iota
	numberOfSCORETypes
)

var scoreNameToType = map[string]scoreType{
	"python": pythonSCORE,
}

func (t scoreType) String() string {
	switch t {
	case pythonSCORE:
		return "PythonSCORE"
	default:
		return "UnknownSCORE"
	}
}

type Manager interface {
	Get(t string) Proxy
	Loop() error
}

type manager struct {
	server ipc.Server
	lock   sync.Mutex

	scores [numberOfSCORETypes]struct {
		waitor *sync.Cond
		ready  *proxy
		using  *proxy
	}
}

func (m *manager) OnConnect(c ipc.Connection) error {
	if proxy, err := newConnection(m, c); err != nil {
		go proxy.HandleMessages()
		return nil
	} else {
		return err
	}
}

func (m *manager) OnClose(c ipc.Connection) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 0; i < len(m.scores); i++ {
		e := &m.scores[i]
		for p := e.ready; p != nil; p = p.next {
			if p.conn == c {
				m.detach(p)
				return nil
			}
		}
		for p := e.using; p != nil; p = p.next {
			if p.conn == c {
				m.detach(p)
				return nil
			}
		}
	}
	return errors.New("NotFound")
}

func (m *manager) onReady(t scoreType, p *proxy) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if t < 0 || t >= numberOfSCORETypes {
		return
	}

	m.detach(p)
	score := &m.scores[t]
	m.attach(&score.ready, p)
	if p.next == nil {
		score.waitor.Broadcast()
	}
}

func (m *manager) detach(p *proxy) {
	if p.pprev == nil {
		return
	}
	*p.pprev = p.next
	if p.next != nil {
		p.next.pprev = p.pprev
	}
	p.pprev = nil
	p.next = nil
}

func (m *manager) attach(r **proxy, p *proxy) {
	p.next = *r
	if p.next != nil {
		p.next.pprev = &p.next
	}
	p.pprev = r
	*r = p
}

func (m *manager) Get(name string) Proxy {
	t, ok := scoreNameToType[name]
	if !ok {
		return nil
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	score := &m.scores[t]
	for score.ready == nil {
		score.waitor.Wait()
	}
	p := score.ready
	m.detach(p)
	m.attach(&score.using, p)
	return p
}

func (m *manager) Loop() error {
	return m.server.Loop()
}

func New(net, addr string) (*manager, error) {
	srv := ipc.NewServer()
	err := srv.Listen(net, addr)
	if err != nil {
		return nil, err
	}
	m := new(manager)
	srv.SetHandler(m)
	m.server = srv
	for i := 0; i < len(m.scores); i++ {
		m.scores[i].waitor = sync.NewCond(&m.lock)
	}

	return m, nil
}
