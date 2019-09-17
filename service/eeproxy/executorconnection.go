package eeproxy

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/common/log"
)

type executorConnection struct {
	eem *executorManager
	log log.Logger
}

func (eec *executorConnection) deregisterFrom(c ipc.Connection) {
	c.SetHandler(msgVERSION, nil)
	c.SetHandler(managerVERSION, nil)
}

func (eec *executorConnection) registerTo(c ipc.Connection) {
	c.SetHandler(msgVERSION, eec)
	c.SetHandler(managerVERSION, eec)
}

func (eec *executorConnection) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
	switch msg {
	case msgVERSION:
		var m versionMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		eec.deregisterFrom(c)
		return eec.eem.onEEConnect(c, m.Type, m.Version, m.UID)
	case managerVERSION:
		var m managerVersion
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		eec.deregisterFrom(c)
		return eec.eem.onEEMConnect(c, m.Type, m.Version)
	}
	return errors.InvalidStateError.Errorf(
		"Invalid message(%d) before version", msg)
}

func newEEConnection(eem *executorManager, log log.Logger, c ipc.Connection) *executorConnection {
	eec := &executorConnection{
		eem: eem,
		log: log,
	}
	eec.registerTo(c)
	return eec
}
