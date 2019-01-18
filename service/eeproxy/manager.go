package eeproxy

import (
	"log"
	"sync"

	"github.com/icon-project/goloop/common/ipc"
	"github.com/pkg/errors"
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
	SetEngine(t string, e Engine) error
	SetInstances(t string, n int) error
	Loop() error
	Close() error
}

type Engine interface {
	Init(net, addr string) error
	SetInstances(n int) error
	OnAttach(uid string) bool
	Kill(uid string) (bool, error)
}

type manager struct {
	server ipc.Server
	lock   sync.Mutex

	scores [numberOfSCORETypes]struct {
		engine Engine
		target int
		active int
		waiter *sync.Cond
		ready  *proxy
		using  *proxy
	}
}

func (m *manager) SetEngine(t string, e Engine) error {
	scoreType, ok := scoreNameToType[t]
	if !ok {
		return errors.Errorf("IllegalScoreType(t=%s)", t)
	}
	addr := m.server.Addr()
	if err := e.Init(addr.Network(), addr.String()); err != nil {
		return err
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	score := &m.scores[scoreType]
	score.engine = e

	return nil
}

func (m *manager) SetInstances(t string, n int) error {
	scoreType, ok := scoreNameToType[t]
	if !ok {
		return errors.Errorf("IllegalScoreType(t=%s)", t)
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	score := &m.scores[scoreType]
	if err := score.engine.SetInstances(n); err != nil {
		return err
	}

	score.target = n
	for score.ready != nil && score.active > score.target {
		item := score.ready
		m.detach(item)
		item.Close()
		score.active -= 1
	}

	return nil
}

func (m *manager) OnConnect(c ipc.Connection) error {
	_, err := newConnection(m, c)
	return err
}

func (m *manager) OnClose(c ipc.Connection) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 0; i < len(m.scores); i++ {
		e := &m.scores[i]
		for p := e.ready; p != nil; p = p.next {
			if p.conn == c {
				m.detach(p)
				e.active -= 1
				return nil
			}
		}
		for p := e.using; p != nil; p = p.next {
			if p.conn == c {
				m.detach(p)
				e.active -= 1
				return nil
			}
		}
	}
	return errors.New("NotFound")
}

func (m *manager) onReady(t scoreType, p *proxy) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if t < 0 || t >= numberOfSCORETypes {
		return errors.Errorf("IllegalScoreType(type=%d)", t)
	}
	score := &m.scores[t]
	if !score.engine.OnAttach(p.uid) {
		return errors.Errorf("InvalidUID(uid=%s)", p.uid)
	}

	if detached := m.detach(p); detached {
		if score.active > score.target {
			score.active -= 1
			return errors.Errorf("ScalingDown(target=%d,active=%d)",
				score.target, score.active)
		}
	} else {
		if score.active >= score.target {
			return errors.Errorf("NoMoreInstance(target=%d,active=%d)",
				score.target, score.active)
		} else {
			score.active += 1
		}
	}

	m.attach(&score.ready, p)
	if p.next == nil {
		score.waiter.Broadcast()
	}
	return nil
}

func (m *manager) detach(p *proxy) bool {
	if p.pprev == nil {
		return false
	}
	*p.pprev = p.next
	if p.next != nil {
		p.next.pprev = p.pprev
	}
	p.pprev = nil
	p.next = nil
	return true
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
		score.waiter.Wait()
	}
	p := score.ready
	if p.reserve() {
		m.detach(p)
		m.attach(&score.using, p)
	}
	return p
}

func (m *manager) Loop() error {
	return m.server.Loop()
}

func (m *manager) Close() error {
	if err := m.server.Close(); err != nil {
		log.Printf("Fail to close IPC server err=%+v", err)
		return err
	}
	// TODO stopping all proxies.
	return nil
}

func (m *manager) Kill(uid string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, s := range m.scores {
		if ok, err := s.engine.Kill(uid); ok {
			return err
		}
	}
	return errors.New("NoEntry")
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
		m.scores[i].waiter = sync.NewCond(&m.lock)
	}

	return m, nil
}
