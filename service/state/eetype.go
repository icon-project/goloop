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

func (e EEType) InstallMethod() string {
	switch e {
	case PythonEE:
		return "on_install"
	case JavaEE:
		return "<init>"
	}
	log.Errorf("UnexpectedEEType(%s)\n", e)
	return ""
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
