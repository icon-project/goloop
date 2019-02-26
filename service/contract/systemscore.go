package contract

import (
	"bytes"
	"reflect"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/state"
)

type SystemScore interface {
	GetAPI() *scoreapi.Info
	Invoke(method string, paramObj *codec.TypedObj) (module.Status, *codec.TypedObj)
}

func CheckMethod(obj SystemScore) bool {
	numMethod := reflect.ValueOf(obj).NumMethod()
	methodInfo := obj.GetAPI()
	for i := 0; i < numMethod; i++ {
		m := reflect.TypeOf(obj).Method(i)
		methodInfo := methodInfo.GetMethod(m.Name)
		if methodInfo == nil {
			continue
		}
		// CHECK INPUT
		numIn := m.Type.NumIn()
		for i := 1; i < numIn; i++ {
			t := m.Type.In(i)
			switch methodInfo.Inputs[i-1].Type {
			case scoreapi.Integer:
				if reflect.TypeOf(int(0)) != t && reflect.TypeOf(int64(0)) != t {
					return false
				}
			case scoreapi.String:
				if reflect.TypeOf(string("")) != t {
					return false
				}
			case scoreapi.Bytes:
				if reflect.TypeOf([]byte{}) != t {
					return false
				}
			case scoreapi.Bool:
				if reflect.TypeOf(bool(false)) != t {
					return false
				}
			case scoreapi.Address:
				if reflect.TypeOf(&common.Address{}).Implements(t) == false {
					return false
				}
			default:
				return false
			}
		}

		numOut := m.Type.NumOut()
		for i := 1; i < numOut; i++ {
			t := m.Type.Out(i)
			switch methodInfo.Outputs[i-1] {
			case scoreapi.Integer:
				if reflect.TypeOf(int(0)) != t && reflect.TypeOf(int64(0)) != t {
					return false
				}
			case scoreapi.String:
				if reflect.TypeOf(string("")) != t {
					return false
				}
			case scoreapi.Bytes:
				if reflect.TypeOf([]byte{}) != t {
					return false
				}
			case scoreapi.Bool:
				if reflect.TypeOf(bool(false)) != t {
					return false
				}
			case scoreapi.Address:
				if reflect.TypeOf(&common.Address{}).Implements(t) == false {
					return false
				}
			default:
				return false
			}
		}
	}
	return true
}

func GetSystemScore(from, to module.Address, cc CallContext) SystemScore {
	// chain score
	// addOn score - static, dynamic
	if bytes.Equal(to.ID(), state.SystemID) == true {
		return &ChainScore{from, to, cc}
	}
	// get account for to
	// get & load so
	// get instance for it.
	return nil
}
