package contract

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-errors/errors"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/state"
)

const (
	FUNC_PREFIX = "Ex_"
)

type SystemScore interface {
	Install(param []byte) error
	Update(param []byte) error
	GetAPI() *scoreapi.Info
	Invoke(method string, paramObj *codec.TypedObj) (module.Status, *codec.TypedObj)
}

func CheckMethod(obj SystemScore) error {
	numMethod := reflect.ValueOf(obj).NumMethod()
	methodInfo := obj.GetAPI()
	invalid := false
	for i := 0; i < numMethod; i++ {
		m := reflect.TypeOf(obj).Method(i)
		if strings.HasPrefix(m.Name, FUNC_PREFIX) == false {
			continue
		}
		mName := strings.TrimPrefix(m.Name, FUNC_PREFIX)
		methodInfo := methodInfo.GetMethod(mName)
		if methodInfo == nil {
			continue
		}
		// CHECK INPUT
		numIn := m.Type.NumIn()
		if len(methodInfo.Inputs) != numIn-1 { //min receiver param
			return errors.New(fmt.Sprintf("Wrong method intput. method[%s]\n", mName))
		}
		var t reflect.Type
		for j := 1; j < numIn; j++ {
			t = m.Type.In(j)
			switch methodInfo.Inputs[j-1].Type {
			case scoreapi.Integer:
				if reflect.TypeOf(&common.HexInt{}) != t {
					invalid = true
				}
			case scoreapi.String:
				if reflect.TypeOf(string("")) != t {
					invalid = true
				}
			case scoreapi.Bytes:
				if reflect.TypeOf([]byte{}) != t {
					invalid = true
				}
			case scoreapi.Bool:
				if reflect.TypeOf(bool(false)) != t {
					invalid = true
				}
			case scoreapi.Address:
				if reflect.TypeOf(&common.Address{}).Implements(t) == false {
					invalid = true
				}
			default:
				invalid = true
			}
			if invalid == true {
				return errors.New(fmt.Sprintf("Wrong system score signature. method : %s, "+
					"expected input[%d] : %v BUT real type : %v", mName, j-1, methodInfo.Inputs[j-1].Type, t))
			}
		}

		numOut := m.Type.NumOut()
		if len(methodInfo.Outputs) != numOut-1 { // minus error
			return errors.New(fmt.Sprintf("Wrong method output. method[%s]\n", mName))
		}
		for j := 0; j < len(methodInfo.Outputs); j++ {
			t := m.Type.Out(j)
			switch methodInfo.Outputs[j] {
			case scoreapi.Integer:
				if reflect.TypeOf(int(0)) != t && reflect.TypeOf(int64(0)) != t {
					invalid = true
				}
			case scoreapi.String:
				if reflect.TypeOf(string("")) != t {
					invalid = true
				}
			case scoreapi.Bytes:
				if reflect.TypeOf([]byte{}) != t {
					invalid = true
				}
			case scoreapi.Bool:
				if reflect.TypeOf(bool(false)) != t {
					invalid = true
				}
			case scoreapi.Address:
				if reflect.TypeOf(&common.Address{}).Implements(t) == false {
					invalid = true
				}
			case scoreapi.List:
				if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
					invalid = true
				}
			case scoreapi.Dict:
				if t.Kind() != reflect.Map {
					invalid = true
				}
			default:
				invalid = true
			}
			if invalid == true {
				return errors.New(fmt.Sprintf("Wrong system score signature. method : %s, "+
					"expected output[%d] : %v BUT real type : %v", mName, j, methodInfo.Outputs[j], t))
			}
		}
	}
	return nil
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
