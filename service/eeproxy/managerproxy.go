package eeproxy

import (
	"sync"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/common/log"
)

const (
	managerVERSION = 100
	managerRUN     = 101
	managerKILL    = 102
	managerEND     = 103
)

type managerVersion struct {
	Version uint16
	Type    string
}

type ManagerProxy interface {
	Run(uuid string) error
	Kill(uuid string) error
}

type managerProxy struct {
	lock    sync.Mutex
	conn    ipc.Connection
	version uint16
	engine  Engine
	log     log.Logger
}

func (e *managerProxy) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
	switch msg {
	case managerEND:
		var uid string
		if _, err := codec.MP.UnmarshalFromBytes(data, &uid); err != nil {
			return err
		}
		e.log.Debug("managerProxy.HandleMessage managerEND")
		e.engine.OnEnd(uid)
	}
	return nil
}

func (e *managerProxy) Run(uuid string) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.conn.Send(managerRUN, &uuid)
}

func (e *managerProxy) Kill(uuid string) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.conn.Send(managerKILL, &uuid)
}

func newManagerProxy(v uint16, c ipc.Connection, e Engine, log log.Logger) (ManagerProxy, error) {
	ep := &managerProxy{
		version: v,
		conn:    c,
		engine:  e,
		log:     log,
	}
	c.SetHandler(managerEND, ep)
	return ep, nil
}
