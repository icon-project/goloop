package eeproxy

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/common/log"
)

const (
	JavaEE = "javaee"
)

const OperationTimeout = 3*time.Second

type javaInstance struct {
	uid    string
	status InstanceStatus
	timer  *time.Timer
}

type javaExecutionEngine struct {
	lock         sync.Mutex
	managerProxy ManagerProxy
	java         string
	args         []string
	target       int
	instances    map[string]*javaInstance
	net, addr    string
	cmd          *exec.Cmd
	timer        *time.Timer
	out          *io.PipeWriter

	conn   ipc.Connection
	logger log.Logger
}

func (e *javaExecutionEngine) Type() string {
	return "java"
}

func (e *javaExecutionEngine) runEE(uid string) error {
	if e.managerProxy == nil {
		e.logger.Debug("Failed to run JavaEE. managerProxy is nil")
		return errors.ErrInvalidState
	}

	if err := e.managerProxy.Run(uid); err != nil {
		return err
	}
	return nil
}

func (e *javaExecutionEngine) start() error {
	out := e.logger.WriterLevel(log.DebugLevel)
	e.cmd = e.newCmd(out, out)
	if err := e.cmd.Start(); err != nil {
		e.logger.Error("Failed to start JAVA EEManager")
		out.Close()
		return err
	}
	e.out = out
	e.timer = time.AfterFunc(time.Second*10, func() {
		e.logger.Panic("Failed to execute Execution Engine Manager")
	})
	e.logger.Debugf("start JavaEE addr(%s), PID(%d), state(%p), \n", e.addr, e.cmd.Process.Pid, e.cmd.ProcessState)
	return nil
}

func (e *javaExecutionEngine) term(i *javaInstance) {
	if i.timer != nil {
		i.timer.Stop()
		i.timer = nil
	}
	delete(e.instances, i.uid)
}

func (e *javaExecutionEngine) Init(net, addr string) error {
	e.net = net
	abs, err := filepath.Abs(addr)
	if err != nil {
		e.logger.Errorf("Failed to convert abs from(%s), err(%s)\n", addr, err)
	}
	e.addr = abs
	if e.managerProxy != nil {
		return errors.ErrInvalidState
	}
	e.logger.Debugf("JavaEE Init net(%s), addr(%s)\n", net, e.addr)
	if err := e.start(); err != nil {
		return err
	}
	return nil
}

func (e *javaExecutionEngine) OnRunTimeout(uid string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if is, ok := e.instances[uid]; ok && is.status == instanceStarted {
		is.timer = nil
		e.logger.Warnf("TIMEOUT after javaee run(uid=%s)", uid)
		_ = e.conn.Close()
	}
}

func (e *javaExecutionEngine) runInstances() error {
	e.logger.Debugf("runInstances e.target(%d), e.instances(%d)\n", e.target, len(e.instances))
	for e.target > len(e.instances) {
		uid := newUID()
		e.logger.Debugf("runInstances with uid(%s)\n", uid)
		if err := e.runEE(uid); err != nil {
			log.Errorf("Fail to start execution engine err=%+v", err)
			return err
		}
		timer := time.AfterFunc(OperationTimeout, func() {
			e.OnRunTimeout(uid)
		})
		e.instances[uid] = &javaInstance{uid, instanceStarted, timer}
	}

	return nil
}

func (e *javaExecutionEngine) SetInstances(n int) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if n < 0 {
		return errors.ErrIllegalArgument
	}
	e.target = n
	if e.managerProxy == nil {
		return nil
	}

	return e.runInstances()
}

func (e *javaExecutionEngine) OnAttach(uid string) bool {
	e.logger.Debugf("OnAttach uid(%s)\n", uid)
	e.lock.Lock()
	defer e.lock.Unlock()

	if is, ok := e.instances[uid]; ok {
		if is.status == instanceStarted {
			is.timer.Stop()
			is.timer = nil
		}
		is.status = instanceOnline
		return true
	}
	e.logger.Debugf("Invalid UID(%s)\n", uid)
	return false
}

func (e *javaExecutionEngine) OnKillTimeout(uid string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if is, ok := e.instances[uid]; ok {
		is.timer = nil
		e.logger.Warnf("TIMEOUT after javaee kill(uid=%s)", uid)
		_ = e.conn.Close()
	}
}

// restart EE
func (e *javaExecutionEngine) Kill(uid string) (bool, error) {
	e.logger.Debugf("Kill uid(%s)\n", uid)
	e.lock.Lock()
	defer e.lock.Unlock()

	if is, ok := e.instances[uid]; ok {
		if err := e.managerProxy.Kill(is.uid); err != nil {
			return true, err
		}
		is.timer = time.AfterFunc(OperationTimeout, func() {
			e.OnKillTimeout(uid)
		})
		return true, nil
	}
	return false, nil
}

func (e *javaExecutionEngine) OnConnect(conn ipc.Connection, version uint16) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	var err error
	if e.conn != nil && e.managerProxy != nil {
		// TODO define error
		e.logger.Errorf("Failed to connect\n")
		return nil
	}
	e.timer.Stop()

	e.conn = conn
	e.managerProxy, err = newManagerProxy(version, conn, e, e.logger)
	if err == nil {
		err = e.runInstances()
	}
	return err
}

// OnEnd is called when executor is terminated
func (e *javaExecutionEngine) OnEnd(uid string) bool {
	e.lock.Lock()
	defer e.lock.Unlock()

	if is, ok := e.instances[uid]; ok {
		e.logger.Infof("OnEnd uid(%s) status(%s)\n", uid, is.status)
		if is.status == instanceOnline {
			e.term(is)
			if err := e.runInstances(); err != nil {
				return false
			}
			return true
		} else {
			// if the proxy with the uid is not connected, do not retry run
			e.logger.Debugf("Failed to start executor(%s)\n", uid)
		}
	} else {
		e.logger.Debugf("Invalid UID(%s)\n", uid)
	}
	return false
}

// OnClose is called when executor manager is terminated
func (e *javaExecutionEngine) OnClose(conn ipc.Connection) bool {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.conn != conn {
		return false
	}
	// running
	if e.cmd.ProcessState == nil {
		if err := e.cmd.Process.Kill(); err != nil {
			e.logger.Warnf("Failed to kill Java EEManager. err(%s), pid(%d)\n", err, e.cmd.Process.Pid)
		}
		e.cmd.Process.Wait()
	}
	e.out.Close()
	e.conn = nil
	e.managerProxy = nil

	for _, i := range e.instances {
		e.term(i)
	}

	if err := e.start(); err != nil {
		e.logger.Panicf("Failed to start Java EEManager. err(%s)\n", err)
	}
	return true
}

func (e *javaExecutionEngine) newCmd(stdout, stderr io.WriteCloser) *exec.Cmd {
	args := append(e.args, e.addr)
	cmd := exec.Command(e.java, args...)
	logLevel := "JAVAEE_LOG_LEVEL=" + e.logger.GetLevel().String()
	cmd.Env = append([]string{logLevel}, os.Environ()...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd
}

func NewJavaEE(logger log.Logger) (Engine, error) {
	binPath, ok := os.LookupEnv("JAVAEE_BIN")
	if !ok {
		return nil, errors.IllegalArgumentError.Errorf("JAVAEE_BIN not set!")
	}
	var e javaExecutionEngine
	e.instances = make(map[string]*javaInstance)
	e.java = "/bin/sh"
	e.args = []string{binPath}
	e.logger = logger.WithFields(log.Fields{log.FieldKeyModule: JavaEE})
	return &e, nil
}
