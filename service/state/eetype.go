package state

import (
	"fmt"
	"sort"
	"strings"

	"github.com/icon-project/goloop/common/errors"
)

type EEType string

const (
	NullEE   EEType = ""
	PythonEE EEType = "python"
	JavaEE   EEType = "java"
	SystemEE EEType = "system"
)

const (
	AllEETypeString = "*"
)

var (
	installMethods = map[EEType]string{
		PythonEE: "on_install",
		JavaEE:   "<init>",
		SystemEE: "<Install>",
	}
	updateMethods = map[EEType]string{
		PythonEE: "on_update",
		JavaEE:   "<init>",
		SystemEE: "<Update>",
	}
	allowUpdateFromTo = map[EEType]map[EEType]bool{
		PythonEE: {
			PythonEE: true,
			JavaEE: true,
		},
		JavaEE: {
			JavaEE: true,
		},
	}
	needAudit = map[EEType]bool{
		PythonEE: true,
	}
)

func (e EEType) InstallMethod() (string, bool) {
	if method, ok := installMethods[e]; ok {
		return method, true
	}
	return "", false
}

func (e EEType) UpdateMethod(from EEType) (string, bool) {
	if allowTo, ok := allowUpdateFromTo[from]; ok {
		if allow, ok := allowTo[e]; ok && allow {
			if method, ok := updateMethods[e]; ok {
				return method, true
			}
		}
	}
	return "", false
}

func (e EEType) AbleToUpdate(org EEType) bool {
	if allowTo, ok := allowUpdateFromTo[org]; ok {
		allow, _ := allowTo[e]
		return allow
	}
	return false
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

func (e EEType) NeedAudit() bool {
	if yn, ok := needAudit[e]; ok {
		return yn
	} else {
		return false
	}
}

func EETypeFromContentType(ct string) (EEType, bool) {
	switch ct {
	case CTAppZip:
		return PythonEE, true
	case CTAppJava:
		return JavaEE, true
	case CTAppSystem:
		return SystemEE, true
	default:
		return NullEE, false
	}
}

func MustEETypeFromContentType(ct string) EEType {
	if et, ok := EETypeFromContentType(ct); !ok {
		panic(fmt.Sprintf("InvalidContentType(type=%s)", ct))
	} else {
		return et
	}
}

func ValidateEEType(et EEType) bool {
	switch et {
	case PythonEE, JavaEE, SystemEE:
		return true
	default:
		return false
	}
}

type EETypes interface {
	Contains(et EEType) bool
	String() string
}

type allEETypes struct{}

func (ts allEETypes) Contains(et EEType) bool {
	return true
}

func (ts allEETypes) String() string {
	return AllEETypeString
}

var AllEETypes EETypes = allEETypes{}

type EETypeFilter map[EEType]bool

func (ets EETypeFilter) Contains(et EEType) bool {
	yn, _ := ets[et]
	return yn
}

func (ets EETypeFilter) String() string {
	keys := make([]string, 0, len(ets))
	for k, _ := range ets {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func ParseEETypes(s string) (EETypes, error) {
	if s == AllEETypeString {
		return AllEETypes, nil
	}
	etf := make(map[EEType]bool)
	if len(s) > 0 {
		ets := strings.Split(s, ",")
		for _, ss := range ets {
			if et := EEType(ss); ValidateEEType(et) {
				etf[et] = true
			} else {
				return nil, errors.IllegalArgumentError.Errorf(
					"InvalidEEType(EEType=%s)", et)
			}
		}
	}
	return EETypeFilter(etf), nil
}
