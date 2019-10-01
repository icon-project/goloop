package contract

import (
	"math/big"
	"reflect"
	"strings"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/icon-project/goloop/service/scoreresult"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
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

func GetSystemScore(contentID string, params ...interface{}) (score SystemScore, err error) {
	defer func() {
		if e := recover(); e != nil {
			if e2, ok := e.(error); ok {
				err = e2
			} else {
				err = errors.CriticalUnknownError.Errorf(
					"Recover from e=%+v", e)
			}
		}
	}()
	v, ok := newSysScore[contentID]
	if ok == false {
		return nil, scoreresult.ContractNotFoundError.Errorf(
			"ContractNotFound(cid=%s)", contentID)
	}

	f := reflect.ValueOf(v)
	fType := f.Type()
	if len(params) != fType.NumIn() {
		return nil, scoreresult.InvalidInstanceError.Errorf(
			"WrongParamNum(req:%d, pass:%d", fType.NumIn(), len(params))
	}

	in := make([]reflect.Value, len(params))
	for i, p := range params {
		pValue := reflect.ValueOf(p)
		if !pValue.IsValid() {
			in[i] = reflect.New(fType.In(i)).Elem()
			continue
		}
		if !pValue.Type().AssignableTo(fType.In(i)) {
			return nil, scoreresult.InvalidInstanceError.Errorf(
				"Can't cast from %s to %s", pValue.Type(), fType.In(i))
		}
		in[i] = reflect.New(fType.In(i)).Elem()
		in[i].Set(pValue)
	}

	result := f.Call(in)

	if len(result) < 1 {
		return nil, scoreresult.InvalidInstanceError.New(
			"Fail to create system score.")
	}

	if result[0].IsNil() {
		return nil, scoreresult.InvalidInstanceError.New(
			"Fail to create system score instance")
	}

	score, ok = result[0].Interface().(SystemScore)
	if ok == false {
		return nil, scoreresult.InvalidInstanceError.Errorf(
			"Not SystemScore. Returned Type is %s", result[0].Type().String())
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
		if len(methodInfo.Inputs) != numIn-1 {
			return scoreresult.InvalidInstanceError.Errorf(
				"Wrong method input. method[%s]", mName)
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
				return scoreresult.InvalidInstanceError.Errorf(
					"wrong system score signature. method : %s, "+
						"expected input[%d] : %v BUT real type : %v", mName, j-1, methodInfo.Inputs[j-1].Type, t)
			}
		}

		numOut := m.Type.NumOut()
		if len(methodInfo.Outputs) != numOut-1 {
			return scoreresult.InvalidInstanceError.Errorf(
				"Wrong method output. method[%s]", mName)
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
				return scoreresult.InvalidInstanceError.Errorf(
					"Wrong system score signature. method : %s, "+
						"expected output[%d] : %v BUT real type : %v", mName, j, methodInfo.Outputs[j], t)
			}
		}
	}
	return nil
}

func Invoke(score SystemScore, method string, paramObj *codec.TypedObj) (status error, result *codec.TypedObj, steps *big.Int) {
	defer func() {
		if err := recover(); err != nil {
			log.Debugf("Fail to sysCall method[%s]. err=%+v\n", method, err)
			status = scoreresult.UnknownFailureError.Errorf("Recover obj=%+v", err)
			result = nil
		}
	}()
	steps = big.NewInt(0)
	m := reflect.ValueOf(score).MethodByName(FUNC_PREFIX + method)
	if m.IsValid() == false {
		return scoreresult.ErrMethodNotFound, nil, steps
	}
	mType := m.Type()

	var params []interface{}
	if ps, err := common.DecodeAny(paramObj); err != nil {
		return scoreresult.ErrInvalidParameter, nil, steps
	} else {
		var ok bool
		params, ok = ps.([]interface{})
		if !ok {
			return scoreresult.ErrInvalidParameter, nil, steps
		}
	}

	if len(params) != mType.NumIn() {
		return scoreresult.ErrInvalidParameter, nil, steps
	}

	objects := make([]reflect.Value, len(params))
	for i, p := range params {
		oType := mType.In(i)
		pValue := reflect.ValueOf(p)
		if !pValue.IsValid() {
			objects[i] = reflect.New(mType.In(i)).Elem()
			continue
		}
		if !pValue.Type().AssignableTo(oType) {
			return scoreresult.ErrInvalidParameter, nil, steps
		}
		objects[i] = reflect.New(mType.In(i)).Elem()
		objects[i].Set(pValue)
	}

	r := m.Call(objects)
	rLen := len(r)

	if rLen > 0 {
		last := r[rLen-1].Interface()
		if last != nil {
			if err, ok := last.(error); ok {
				status = err
			} else {
				status = scoreresult.ErrInvalidInstance
			}
		}
		if rLen == 2 {
			result, status = common.EncodeAny(r[0].Interface())
		}
	}
	return
}

func InstallSystemScore(addr []byte, cid string, param []byte, ctx Context, receipt txresult.Receipt, txHash []byte) error {
	sas := ctx.GetAccountState(addr)
	sas.InitContractAccount(nil)
	sas.DeployContract(nil, "system", state.CTAppSystem,
		nil, nil)
	if err := sas.AcceptContract(txHash, nil); err != nil {
		return err
	}
	sysScore, err := GetSystemScore(cid,
		common.NewContractAddress(addr), NewCallContext(ctx, receipt, false), ctx.Logger())
	if err != nil {
		return err
	}
	if err := sysScore.Install(param); err != nil {
		return err
	}
	if err := CheckMethod(sysScore); err != nil {
		return err
	}
	sas.SetAPIInfo(sysScore.GetAPI())
	return nil
}
