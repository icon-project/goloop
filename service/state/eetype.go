package state

import (
	"github.com/icon-project/goloop/common/log"
)

type EEType string

const (
	PythonEE EEType = "python"
	JavaEE   EEType = "java"
	SystemEE EEType = "system"
)

var (
	installMethods = map[EEType]string{
		PythonEE: "on_install",
		JavaEE:   "<init>",
		SystemEE: "<Install>",
	}
	updateMethods = map[EEType]string{
		PythonEE: "on_update",
		JavaEE:   "",
		SystemEE: "",
	}
)

func (e EEType) InstallMethod() string {
	if method, ok := installMethods[e]; ok {
		return method
	}
	log.Errorf("UnexpectedEEType(%s)\n", e)
	return ""
}

func (e EEType) UpdateMethod() string {
	if method, ok := updateMethods[e]; ok {
		return method
	}
	log.Errorf("UnexpectedEEType(%s)\n", e)
	return ""
}

func (e EEType) IsInternalMethod(s string) bool {
	if method, ok := installMethods[e]; ok {
		if method == s {
			return true
		}
	}
	if method, ok := updateMethods[e]; ok {
		if method == s {
			return true
		}
	}
	return false
}

func (e EEType) String() string {
	return string(e)
}

// Only "application/zip" and "application/java" are allowed as contentType by server validator.
func EETypeFromContentType(ct string) EEType {
	switch ct {
	case CTAppZip:
		return PythonEE
	case CTAppJava:
		return JavaEE
	case CTAppSystem:
		return SystemEE
	default:
		log.Errorf("Unexpected contentType(%s)\n", ct)
		return ""
	}
}
