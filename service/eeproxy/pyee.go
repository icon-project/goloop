package eeproxy

import (
	"fmt"
	"github.com/icon-project/goloop/common/log"
	"os"
	"os/exec"
	"sync"

	"github.com/icon-project/goloop/common/errors"
)

type InstanceStatus int

const (
	instanceStarted InstanceStatus = iota
	instanceOnline
	instanceStopped
	instanceError
)

func (s InstanceStatus) String() string {
	switch s {
	case instanceStarted:
		return "STARTED"
	case instanceOnline:
		return "ONLINE"
	case instanceStopped:
		return "STOPPED"
	case instanceError:
		return "ERROR"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(s))
	}
}

type pythonInstance struct {
	uid    string
	cmd    *exec.Cmd
	status InstanceStatus
}

type pythonExecutionEngine struct {
	lock      sync.Mutex
	python    string
	args      []string
	target    int
	instances map[string]*pythonInstance
	net, addr string
}

func (e *pythonExecutionEngine) Type() string {
	return "python"
}

func (e *pythonExecutionEngine) Kill(uid string) (bool, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if is, ok := e.instances[uid]; ok {
		return true, is.cmd.Process.Kill()
	} else {
		return false, nil
	}
}

func (e *pythonExecutionEngine) Init(net, addr string) error {
	if net != "unix" {
		return errors.Errorf("IllegalNetwork(net=%s)", net)
	}
	e.net = net
	e.addr = addr
	return nil
}

func (e *pythonExecutionEngine) SetInstances(n int) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if n < 0 {
		return errors.ErrIllegalArgument
	}

	e.target = n
	for e.target > len(e.instances) {
		if err := e.start(); err != nil {
			log.Errorf("Fail to start execution engine err=%+v", err)
			return err
		}
	}
	return nil
}

func (e *pythonExecutionEngine) OnAttach(uid string) bool {
	e.lock.Lock()
	defer e.lock.Unlock()

	if is, ok := e.instances[uid]; ok {
		is.status = instanceOnline
		return true
	}
	return false
}

func (e *pythonExecutionEngine) newCmd(uid string) *exec.Cmd {
	args := append(e.args, "-s", e.addr, "-u", uid)
	cmd := exec.Command(e.python, args...)
	cmd.Env = append([]string{"PYTHONPATH=./pyee"}, os.Environ()...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func (e *pythonExecutionEngine) run(is *pythonInstance) {
	for true {
		err := is.cmd.Wait()
		log.Tracef("Wait result uid=%s err=%+v\n", is.uid, err)

		e.lock.Lock()
		if is.status != instanceOnline {
			log.Errorf("Fail to get on-line uid=%s\n", is.uid)
			is.status = instanceError
			e.lock.Unlock()
			return
		}
		if len(e.instances) > e.target {
			log.Tracef("End the instance uid=%s\n", is.uid)
			delete(e.instances, is.uid)
			e.lock.Unlock()
			return
		}

		delete(e.instances, is.uid)
		is.uid = newUID()
		is.cmd = e.newCmd(is.uid)
		e.instances[is.uid] = is

		log.Tracef("Restart the instance uid=%s\n", is.uid)
		if err := is.cmd.Start(); err != nil {
			is.status = instanceError
			log.Errorf("Fail to start engine uid=%s err=%+v",
				is.uid, err)
		} else {
			is.status = instanceStarted
		}
		e.lock.Unlock()
	}
}

func (e *pythonExecutionEngine) start() error {
	uid := newUID()
	cmd := e.newCmd(uid)
	if err := cmd.Start(); err != nil {
		return err
	}
	is := &pythonInstance{
		uid:    uid,
		status: instanceStarted,
		cmd:    cmd,
	}
	e.instances[uid] = is
	go e.run(is)
	return nil
}

func NewPythonEE() (Engine, error) {
	var e pythonExecutionEngine
	e.instances = make(map[string]*pythonInstance)
	e.python = "python3"
	e.args = []string{"-u", "-m", "pyexec"}
	return &e, nil
}
