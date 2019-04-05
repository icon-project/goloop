package eeproxy

import (
	"sync"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/ipc"
)

type RequestPriority int

const (
	ForTransaction RequestPriority = iota
	ForQuery
)
const (
	numberOfPriorities = 2
)

type Manager interface {
	GetExecutor(pr RequestPriority) *Executor
	SetInstances(total, tx, query int) error
	Loop() error
	Close() error
}

type Engine interface {
	Type() string
	Init(net, addr string) error
	SetInstances(n int) error
	OnAttach(uid string) bool
	Kill(uid string) (bool, error)
}

type Executor struct {
	priority RequestPriority
	manager  *executorManager
	typeMap  map[string]int
	proxies  []*proxy
}

func (e *Executor) Get(name string) Proxy {
	t, ok := e.typeMap[name]
	if !ok {
		return nil
	}
	return e.proxies[t]
}

func (e *Executor) Release() {
	e.manager.onRelease(e.priority, e)
	for _, p := range e.proxies {
		p.Release()
	}
}

func (e *Executor) Kill() {
	for _, p := range e.proxies {
		p.Kill()
	}
}

type engine struct {
	engine Engine
	active int
	ready  *proxy
	using  *proxy
}

type executorState struct {
	limit    int
	assigned int
	waiter   *sync.Cond
	waiting  int
}

type executorManager struct {
	lock sync.Mutex

	server ipc.Server

	typeMap map[string]int
	engines []*engine

	executorLimit  int
	executorStates [numberOfPriorities]executorState
}

func (em *executorManager) onReady(t string, p *proxy) error {
	em.lock.Lock()
	defer em.lock.Unlock()

	i, ok := em.typeMap[t]
	if !ok {
		return errors.Errorf("InvalidApplicationType:%s", t)
	}

	e := em.engines[i]

	if !e.engine.OnAttach(p.uid) {
		return errors.Errorf("InvalidUID(uid=%s)", p.uid)
	}

	if p.detach() {
		if e.active > em.executorLimit {
			e.active -= 1
			return errors.Errorf("ScalingDown(target=%d,active=%d)",
				em.executorLimit, e.active)
		}
	} else {
		e.active += 1
	}
	p.attachTo(&e.ready)

	for i := range em.executorStates {
		s := em.executorStates[i]
		if s.assigned < s.limit && s.waiting > 0 {
			s.waiter.Signal()
			return nil
		}
	}
	return nil
}

func (em *executorManager) kill(u string) error {
	for _, e := range em.engines {
		if ok, err := e.engine.Kill(u); ok {
			return err
		}
	}
	return errors.New("NoEntry")
}

func (em *executorManager) OnConnect(c ipc.Connection) error {
	_, err := newConnection(em, c)
	return err
}

func (em *executorManager) OnClose(c ipc.Connection) error {
	em.lock.Lock()
	defer em.lock.Unlock()

	for _, e := range em.engines {
		for p := e.ready; p != nil; p = p.next {
			if p.conn == c {
				p.detach()
				e.active -= 1
				return nil
			}
		}
		for p := e.using; p != nil; p = p.next {
			if p.conn == c {
				p.detach()
				e.active -= 1
				return nil
			}
		}
	}
	return errors.New("UnknownConnection")
}

func (em *executorManager) Close() error {
	if err := em.server.Close(); err != nil {
		return err
	}
	return nil
}

func (em *executorManager) createExecutorInLock(pr RequestPriority) *Executor {
	ps := make([]*proxy, len(em.engines))
	for i, e := range em.engines {
		if e.ready == nil {
			return nil
		}
		ps[i] = e.ready
	}
	for i, p := range ps {
		p.detach()
		p.attachTo(&em.engines[i].using)
	}
	return &Executor{
		priority: pr,
		manager:  em,
		proxies:  ps,
		typeMap:  em.typeMap,
	}
}

func (em *executorManager) onRelease(pr RequestPriority, ex *Executor) {
	em.lock.Lock()
	defer em.lock.Unlock()

	em.executorStates[pr].assigned -= 1
}

func (em *executorManager) GetExecutor(pr RequestPriority) *Executor {
	em.lock.Lock()
	defer em.lock.Unlock()

	es := &em.executorStates[pr]
	es.waiting += 1
	for {
		if es.assigned < es.limit {
			e := em.createExecutorInLock(pr)
			if e != nil {
				es.assigned += 1
				es.waiting -= 1
				return e
			}
		}
		es.waiter.Wait()
	}
}

func (em *executorManager) SetInstances(total, tx, query int) error {
	em.lock.Lock()
	defer em.lock.Unlock()

	for _, e := range em.engines {
		if err := e.engine.SetInstances(total); err != nil {
			return err
		}
	}

	em.executorLimit = total
	em.executorStates[ForTransaction].limit = tx
	em.executorStates[ForQuery].limit = query

	for _, e := range em.engines {
		for e.ready != nil && e.active > em.executorLimit {
			item := e.ready
			item.detach()
			item.close()
			e.active -= 1
		}
	}
	return nil
}

func (em *executorManager) Loop() error {
	return em.server.Loop()
}

func NewManager(net, addr string, engines ...Engine) (Manager, error) {
	srv := ipc.NewServer()
	err := srv.Listen(net, addr)
	if err != nil {
		return nil, err
	}
	im := new(executorManager)
	srv.SetHandler(im)
	im.server = srv

	for i := 0; i < len(im.executorStates); i++ {
		im.executorStates[i].waiter = sync.NewCond(&im.lock)
	}

	im.engines = make([]*engine, len(engines))
	im.typeMap = make(map[string]int)
	for i, e := range engines {
		if err := e.Init(net, addr); err != nil {
			return nil, err
		}
		im.engines[i] = &engine{engine: e}
		im.typeMap[e.Type()] = i
	}
	return im, nil
}
