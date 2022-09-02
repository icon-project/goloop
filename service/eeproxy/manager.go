package eeproxy

import (
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/common/log"
)

type RequestPriority int

const (
	ForTransaction RequestPriority = iota
	ForQuery
)
const (
	numberOfPriorities = 2
)

const (
	errorBase                  = errors.CodeService + 300
	ScaleDownError errors.Code = iota + errorBase
	InvalidUUIDError
	InvalidAppTypeError
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
	OnEnd(uid string) bool
	Kill(uid string) (bool, error)
	OnConnect(conn ipc.Connection, version uint16) error
	OnClose(conn ipc.Connection) bool
}

type Executor struct {
	priority RequestPriority
	manager  *executorManager
	proxies  map[string]*proxy
}

func (e *Executor) Get(name string) Proxy {
	if p, ok := e.proxies[name]; ok {
		return p
	} else {
		return nil
	}
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
	e.Release()
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

	engines map[string]*engine

	executorLimit  int
	executorStates [numberOfPriorities]executorState

	log log.Logger
}

func (em *executorManager) onReady(p *proxy) error {
	em.lock.Lock()
	defer em.lock.Unlock()

	e, ok := em.engines[p.scoreType]
	if !ok {
		em.log.Warnf("InvalidApplicationType(%s)", p.scoreType)
		return InvalidAppTypeError.Errorf("InvalidApplicationType:%s", p.scoreType)
	}

	if p.detach() {
		if e.active > em.executorLimit {
			e.active -= 1
			em.log.Infof("Stop proxy=%s-%s (target=%d,active=%d)",
				p.scoreType, p.uid, em.executorLimit, e.active)
			return ScaleDownError.Errorf("ScalingDown(target=%d,active=%d)",
				em.executorLimit, e.active)
		}
	} else {
		if !e.engine.OnAttach(p.uid) {
			em.log.Warnf("InvalidUUID(uid=%s)", p.uid)
			return InvalidUUIDError.Errorf("InvalidUID(uid=%s)", p.uid)
		}
		e.active += 1
	}
	p.attachTo(&e.ready)

	for i := range em.executorStates {
		s := em.executorStates[i]
		if s.assigned < s.limit && s.waiting > 0 {
			s.waiter.Signal()
			break
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
	_ = newEEConnection(em, em.log, c)
	return nil
}

func (em *executorManager) OnClose(c ipc.Connection) {
	l := common.LockForAutoCall(&em.lock)
	defer l.Unlock()

	for _, e := range em.engines {
		for p := e.ready; p != nil; p = p.next {
			if p.conn == c {
				p.detach()
				e.active -= 1
				return
			}
		}
		for p := e.using; p != nil; p = p.next {
			if p.conn == c {
				p.detach()
				l.CallAfterUnlock(func() {
					p.OnClose()
				})
				e.active -= 1
				return
			}
		}
		if e.engine.OnClose(c) {
			return
		}
	}
}

func (em *executorManager) Close() error {
	if err := em.server.Close(); err != nil {
		return err
	}
	return nil
}

func (em *executorManager) createExecutorInLock(pr RequestPriority) *Executor {
	ps := make(map[string]*proxy)
	for name, e := range em.engines {
		if e.ready == nil {
			return nil
		}
		ps[name] = e.ready
	}
	for i, p := range ps {
		p.detach()
		p.attachTo(&em.engines[i].using)
		p.reserve()
	}
	return &Executor{
		priority: pr,
		manager:  em,
		proxies:  ps,
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
			em.log.Infof("Stop proxy=%s-%s (active=%d > limit=%d)",
				e.engine.Type(), e.ready.uid,
				e.active, em.executorLimit)
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

// onEEMConnect handle a connection from Execution Environment Manager
func (em *executorManager) onEEMConnect(conn ipc.Connection, t string, v uint16) error {
	em.log.Infof("ExecutorManager.onEEMConnect(type=%s,version=%d)", t, v)
	em.lock.Lock()
	defer em.lock.Unlock()

	for _, e := range em.engines {
		if e.engine.Type() == t {
			return e.engine.OnConnect(conn, v)
		}
	}
	return errors.NotFoundError.Errorf("UnknownType(%s)", t)
}

// onEEConnect handle a connection from Execution Environment
func (em *executorManager) onEEConnect(conn ipc.Connection, t string, v uint16, uid string) error {
	em.log.Infof("ExecutorManager.onEEConnect(type=%s,version=%d,uid=%s)", t, v, uid)
	if _, err := newProxy(em, conn, em.log, t, v, uid); err != nil {
		return err
	}
	return nil
}

func NewManager(net, addr string, l log.Logger, engines ...Engine) (Manager, error) {
	srv := ipc.NewServer()
	err := srv.Listen(net, addr)
	if err != nil {
		return nil, err
	}

	em := new(executorManager)
	srv.SetHandler(em)
	em.server = srv
	em.log = l.WithFields(log.Fields{log.FieldKeyModule: "EEP"})

	for i := 0; i < len(em.executorStates); i++ {
		em.executorStates[i].waiter = sync.NewCond(&em.lock)
	}

	em.engines = make(map[string]*engine)
	for _, e := range engines {
		if err := e.Init(net, addr); err != nil {
			return nil, err
		}
		em.engines[e.Type()] = &engine{engine: e}
	}
	return em, nil
}
