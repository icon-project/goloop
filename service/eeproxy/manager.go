package eeproxy

import (
	"github.com/icon-project/goloop/common/ipc"
	"github.com/pkg/errors"
	"sync"
)

type manager struct {
	server ipc.Server
	lock   sync.Mutex
	waitor *sync.Cond

	ready *proxy
	using *proxy
}

func (m *manager) OnConnect(c ipc.Connection) error {
	if proxy, err := newConnection(c); err != nil {
		m.lock.Lock()
		defer m.lock.Unlock()

		m.attach(&m.ready, proxy)
		if proxy.next == nil {
			m.waitor.Signal()
		}
		return nil
	} else {
		return err
	}
}

func (m *manager) OnClose(c ipc.Connection) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for p := m.ready; p != nil; p = p.next {
		if p.conn == c {
			m.detach(p)
			return nil
		}
	}
	return errors.New("NotFound")
}

func (m *manager) detach(p *proxy) {
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

func (m *manager) Get() Proxy {
	m.lock.Lock()
	defer m.lock.Unlock()

	for m.ready == nil {
		m.waitor.Wait()
	}
	p := m.ready
	m.detach(p)
	m.attach(&m.using, p)
	return p
}

func (m *manager) Release(p Proxy) {
	proxy := p.(*proxy)
	m.lock.Lock()
	defer m.lock.Unlock()

	m.detach(proxy)
	m.attach(&m.ready, proxy)
	if proxy.next == nil {
		m.waitor.Signal()
	}
}

func (m *manager) Loop() error {
	return m.server.Loop()
}

func New() (*manager, error) {
	srv := ipc.NewServer()
	err := srv.Listen("unix", "/tmp/execution_engine")
	if err != nil {
		return nil, err
	}
	m := new(manager)
	srv.SetHandler(m)
	m.server = srv
	m.waitor = sync.NewCond(&m.lock)

	return m, nil
}
