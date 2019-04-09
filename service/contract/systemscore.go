package contract

import (
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strings"

	"github.com/go-errors/errors"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
)

const (
	FUNC_PREFIX = "Ex_"
)

const (
	CID_CHAIN = "CID_CHAINSCORE"
)

var newSysScore = map[string]interface{}{
	CID_CHAIN: NewChainScore,
}

type SystemScore interface {
	Install(param []byte) error
	Update(param []byte) error
	GetAPI() *scoreapi.Info
}

func GetSystemScore(contendId string, params ...interface{}) (score SystemScore, err error) {
	defer func() {
		if e := recover(); e != nil {
			msg := fmt.Sprintf("Failed to call function for %s : err  = %s\n", contendId, e)
			log.Printf(msg)
			err = errors.New(msg)
		}
	}()
	v, ok := newSysScore[contendId]
	if ok == false {
		return nil, errors.New(fmt.Sprintf("Wrong contentId. %s\n", contendId))
	}
	f := reflect.ValueOf(v)
	if len(params) != f.Type().NumIn() {
		return nil, errors.New("Wrong params number.")
	}
	in := make([]reflect.Value, len(params))
	for i, p := range params {
		in[i] = reflect.ValueOf(p)
	}
	result := f.Call(in)

	if result[0].IsNil() {
		return nil, errors.New("Failed to create new systemScore")
	}

	score, ok = result[0].Interface().(SystemScore)
	if ok == false {
		return nil, errors.New(fmt.Sprintf("Not SystemScore. Retuned Type is %s", result[0].Type().String()))
	}
	return score, nil
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

func Invoke(score SystemScore, method string, paramObj *codec.TypedObj) (status module.Status, result *codec.TypedObj, steps *big.Int) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Failed to sysCall method[%s]. err = %s\n", method, err)
			status = module.StatusSystemError
		}
	}()
	steps = big.NewInt(0)
	m := reflect.ValueOf(score).MethodByName(FUNC_PREFIX + method)
	if m.IsValid() == false {
		return module.StatusMethodNotFound, nil, steps
	}
	params, _ := common.DecodeAny(paramObj)
	numIn := m.Type().NumIn()
	objects := make([]reflect.Value, numIn)
	if l, ok := params.([]interface{}); ok == true {
		if len(l) != numIn {
			return module.StatusInvalidParameter, nil, steps
		}
		for i, v := range l {
			objects[i] = reflect.ValueOf(v)
		}
	}
	// check if it is eventLog or not.
	// if eventLog then cc.AddLog().
	r := m.Call(objects)
	resultLen := len(r)
	var output interface{}

	// last output type in chain score method is error.
	status = module.StatusSuccess
	for i, v := range r {
		if i+1 == resultLen { // last output
			if err := v.Interface(); err != nil {
				log.Printf("Failed to invoke %s on chain score. %s\n", method, err.(error))
				status = module.StatusSystemError
			}
			continue
		} else {
			output = v.Interface()
		}
	}

	result, _ = common.EncodeAny(output)
	// TODO apply used step
	return status, result, steps
}
