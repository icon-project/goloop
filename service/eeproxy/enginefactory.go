package eeproxy

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
)

func AllocEngines(l log.Logger, names ...string) ([]Engine, error) {
	l.Infof("Allocate Engines:%s", names)
	engines := make([]Engine, len(names))
	for i, name := range names {
		switch name {
		case "python":
			if engine, err := NewPythonEE(l); err != nil {
				return nil, err
			} else {
				engines[i] = engine
			}
		case "java":
			if engine, err := NewJavaEE(l); err != nil {
				return nil, err
			} else {
				engines[i] = engine
			}
		default:
			return nil, errors.IllegalArgumentError.Errorf(
				"IllegalEngineName(name=%s)", name)
		}
	}
	return engines, nil
}
