package eeproxy

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/common/log"
)

type InstanceStatus int

const (
	instanceStarted InstanceStatus = iota
	instanceOnline
	instanceStopped
	instanceError
)

const (
	PythonEE = "pyee"
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
	out    io.WriteCloser
}

type pythonExecutionEngine struct {
	lock      sync.Mutex
	python    string
	args      []string
	target    int
	instances map[string]*pythonInstance
	net, addr string
	logger    log.Logger
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
		if err := e.startNew(); err != nil {
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

func (e *pythonExecutionEngine) OnEnd(uid string) bool {
	return true
}

func (e *pythonExecutionEngine) newCmd(uid string, stdout, stderr io.WriteCloser) *exec.Cmd {
	args := append(e.args, "-s", e.addr, "-u", uid)
	cmd := exec.Command(e.python, args...)
	cmd.Env = append([]string{"PYTHONPATH=./pyee"}, os.Environ()...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd
}

func (e *pythonExecutionEngine) run(is *pythonInstance) {
	for true {
		err := is.cmd.Wait()
		e.logger.Tracef("Wait result uid=%s err=%+v\n", is.uid, err)

		e.lock.Lock()
		if is.status != instanceOnline {
			e.logger.Warnf("It's not correctly started status=%s err=%+v",
				is.status, err)
			e.term(is)
			e.lock.Unlock()
			return
		}
		if len(e.instances) > e.target {
			e.logger.Tracef("End the instance uid=%s\n", is.uid)
			e.term(is)
			e.lock.Unlock()
			return
		}
		e.logger.Warnf("Instance uid=%s is killed err=%+v",
			is.uid, err)
		e.term(is)

		e.init(is)
		if err := e.start(is); err != nil {
			e.logger.Errorf("Fail to start instance err=%+v", err)
			e.term(is)
			e.lock.Unlock()
			return
		}
		e.lock.Unlock()
	}
}

func (e *pythonExecutionEngine) init(i *pythonInstance) {
	i.uid = newUID()
	i.out = e.logger.WithFields(log.Fields{
		log.FieldKeyEID: i.uid,
	}).WriterLevel(log.DebugLevel)
	e.instances[i.uid] = i
	i.cmd = e.newCmd(i.uid, i.out, i.out)
	i.status = instanceStopped
}

func (e *pythonExecutionEngine) start(i *pythonInstance) error {
	e.logger.Infof("start instance uid=%s", i.uid)
	if err := i.cmd.Start(); err != nil {
		return err
	}
	i.status = instanceStarted
	return nil
}

func (e *pythonExecutionEngine) term(i *pythonInstance) {
	_ = i.out.Close()
	delete(e.instances, i.uid)
}

func (e *pythonExecutionEngine) startNew() error {
	i := new(pythonInstance)
	e.init(i)
	if err := e.start(i); err != nil {
		e.term(i)
		return err
	}
	go e.run(i)
	return nil
}

func (e *pythonExecutionEngine) OnConnect(conn ipc.Connection, version uint16) error {
	return common.ErrUnsupported
}

func (e *pythonExecutionEngine) OnClose(conn ipc.Connection) bool {
	return false
}

func NewPythonEE(logger log.Logger) (Engine, error) {
	var e pythonExecutionEngine
	e.instances = make(map[string]*pythonInstance)
	e.python = "python3"
	e.args = []string{"-u", "-m", "pyexec"}
	e.logger = logger.WithFields(log.Fields{log.FieldKeyModule: PythonEE})
	lv := logger.GetLevel()
	e.args = append(e.args, "-d", lv.String())
	verify, ok := os.LookupEnv("PYEE_VERIFY_PACKAGE")
	if ok && verify == "true" {
		e.args = append(e.args, "-p")
	}
	return &e, nil
}
