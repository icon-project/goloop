package eeproxy

import (
	"container/list"
	"log"
	"os/exec"
	"sync"

	"github.com/pkg/errors"
)

type pythonExecutionEngine struct {
	lock      sync.Mutex
	python    string
	args      []string
	target    int
	instances list.List
	net, addr string
}

func (e *pythonExecutionEngine) Init(net, addr string) error {
	e.net = net
	e.addr = addr
	return nil
}

func (e *pythonExecutionEngine) SetInstances(n int) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if n < 0 {
		return errors.New("IllegalArgument")
	}

	e.target = n
	for e.target > e.instances.Len() {
		if err := e.start(); err != nil {
			log.Fatalf("Fail to start execution engine err=%+v", err)
			return err
		}
	}
	return nil
}

func (e *pythonExecutionEngine) newCmd() *exec.Cmd {
	args := make([]string, len(e.args), len(e.args)+2)
	copy(args, e.args)
	args = append(args, "-s", e.addr)
	return exec.Command(e.python, args...)
}

func (e *pythonExecutionEngine) run(item *list.Element) {
	cmd := item.Value.(*exec.Cmd)
	for true {
		err := cmd.Wait()
		log.Printf("Wait result err=%+v\n", err)
		e.lock.Lock()
		if e.instances.Len() > e.target {
			log.Printf("End the instance\n")
			e.instances.Remove(item)
			e.lock.Unlock()
			return
		}
		cmd = e.newCmd()
		item.Value = cmd
		e.lock.Unlock()
		log.Printf("Restart the instance\n")
		if err := cmd.Start(); err != nil {
			log.Fatalf("Fail to start execution engine err=%+v", err)
		}
	}
}

func (e *pythonExecutionEngine) start() error {
	cmd := e.newCmd()
	if err := cmd.Start(); err != nil {
		return err
	}

	item := e.instances.PushBack(cmd)
	go e.run(item)
	return nil
}

func NewPythonEE() (Engine, error) {
	var e pythonExecutionEngine
	e.instances.Init()
	e.python = "python3"
	e.args = []string{"-m", "pyexec"}
	return &e, nil
}
