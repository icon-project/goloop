package contract

import (
	"math/big"
	"reflect"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	FUNC_PREFIX = "Ex_"
)

const (
	CID_CHAIN = ""
)

type SystemScoreModule struct {
	New func(cid string, cc CallContext, from module.Address, value *big.Int) (SystemScore, error)
}

var systemScoreModules = map[string]*SystemScoreModule{}

func RegisterSystemScore(id string, m *SystemScoreModule) {
	systemScoreModules[id] = m
}

type SystemScore interface {
	Install(param []byte) error
	Update(param []byte) error
	GetAPI() *scoreapi.Info
}

func getSystemScore(contentID string, cc CallContext, from module.Address, value *big.Int) (score SystemScore, err error) {
	v, ok := systemScoreModules[contentID]
	if ok == false {
		return nil, scoreresult.ContractNotFoundError.Errorf(
			"ContractNotFound(cid=%s)", contentID)
	}
	return v.New(contentID, cc, from, value)
}

func CheckStruct(t reflect.Type, fields []scoreapi.Field) error {
	if t.Kind() != reflect.Map {
		return scoreresult.IllegalFormatError.Errorf("NotMapType(%s)", t)
	}
	if t.Key().Kind() != reflect.String {
		return scoreresult.IllegalFormatError.Errorf("KeyTypeInvalid(%s)", t.Key())
	}
	et := t.Elem()
	if et.Kind() != reflect.Interface || et.NumMethod() != 0 {
		return scoreresult.IllegalFormatError.Errorf("ValueTypeInvalid(%s)", et)
	}
	return nil
}

func CheckType(t reflect.Type, mt scoreapi.DataType, fields []scoreapi.Field) error {
	for i := mt.ListDepth(); i > 0; i-- {
		if t.Kind() != reflect.Slice {
			return scoreresult.IllegalFormatError.Errorf("NotCompatibleType(%s)", t)
		}
		t = t.Elem()
	}
	if mt.IsList() {
		if t.Kind() != reflect.Interface || t.NumMethod() != 0 {
			return scoreresult.InvalidParameterError.Errorf("NotCompatible(exp=interface{},type=%s)", t)
		}
		return nil
	}
	switch mt.Tag() {
	case scoreapi.TInteger:
		if ptrOfHexIntType.AssignableTo(t) {
			return nil
		}
		if ptrOfBigIntType.AssignableTo(t) {
			return nil
		}
	case scoreapi.TString:
		switch t.Kind() {
		case reflect.String:
			return nil
		case reflect.Ptr:
			if t.Elem().Kind() == reflect.String {
				return nil
			}
		}
	case scoreapi.TBytes:
		if reflect.TypeOf([]byte{}) == t {
			return nil
		}
	case scoreapi.TBool:
		if reflect.TypeOf(bool(false)) == t {
			return nil
		}
	case scoreapi.TAddress:
		if reflect.TypeOf(&common.Address{}).Implements(t) {
			return nil
		}
	case scoreapi.TStruct:
		if err := CheckStruct(t, fields); err == nil {
			return nil
		} else {
			return scoreresult.IllegalFormatError.Wrapf(err, "NotCompatibleType(%s)", t)
		}
	default:
		return scoreresult.UnknownFailureError.Errorf("UnknownTypeTag(tag=%#x)", mt.Tag())
	}
	return scoreresult.IllegalFormatError.Errorf("NotCompatibleType(%s)", t)
}

func CheckMethod(obj SystemScore, methodInfo *scoreapi.Info) error {
	numMethod := reflect.ValueOf(obj).NumMethod()
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
			return scoreresult.IllegalFormatError.Errorf(
				"Wrong method input. method[%s]", mName)
		}
		for j := 1; j < numIn; j++ {
			t := m.Type.In(j)
			mt := methodInfo.Inputs[j-1].Type
			mf := methodInfo.Inputs[j-1].Fields
			if err := CheckType(t, mt, mf); err != nil {
				return scoreresult.IllegalFormatError.Wrapf(err,
					"wrong system score signature. method : %s, "+
						"expected input[%d] : %v BUT real type : %v", mName, j-1, mt, t)
			}
		}

		numOut := m.Type.NumOut()
		if len(methodInfo.Outputs) != numOut-1 {
			return scoreresult.IllegalFormatError.Errorf(
				"Wrong method output. method[%s]", mName)
		}
		for j := 0; j < len(methodInfo.Outputs); j++ {
			t := m.Type.Out(j)
			switch methodInfo.Outputs[j] {
			case scoreapi.Integer:
				if reflect.TypeOf(int(0)) != t && reflect.TypeOf(int64(0)) != t &&
					ptrOfHexIntType.AssignableTo(t) && ptrOfBigIntType.AssignableTo(t) {
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
				return scoreresult.IllegalFormatError.Errorf(
					"Wrong system score signature. method : %s, "+
						"expected output[%d] : %v BUT real type : %v", mName, j, methodInfo.Outputs[j], t)
			}
		}
	}
	return nil
}

var (
	ptrOfHexIntType  = reflect.TypeOf((*common.HexInt)(nil))
	ptrOfBigIntType  = reflect.TypeOf((*big.Int)(nil))
	sliceOfByteType  = reflect.TypeOf([]byte(nil))
	ptrOfAddressType = reflect.TypeOf((*common.Address)(nil))
)

func AssignHexInt(dstValue reflect.Value, srcValue *common.HexInt) error {
	dstType := dstValue.Type()
	if ptrOfHexIntType.AssignableTo(dstType) {
		dstValue.Set(reflect.ValueOf(srcValue))
		return nil
	}
	if ptrOfBigIntType.AssignableTo(dstType) {
		dstValue.Set(reflect.ValueOf(srcValue.Value()))
		return nil
	}
	dstKind := dstType.Kind()
	switch dstKind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !srcValue.IsUint64() || srcValue.BitLen() > int(dstType.Size()*8) {
			return scoreresult.InvalidParameterError.Errorf("Overflow(type=%s)", dstType)
		}
		dstValue.SetUint(srcValue.Uint64())
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !srcValue.IsInt64() || len(srcValue.Bytes()) > int(dstType.Size()) {
			return scoreresult.InvalidParameterError.Errorf("Overflow(type=%s)", dstType)
		}
		dstValue.SetInt(srcValue.Int64())
		return nil
	case reflect.Bool:
		dstValue.SetBool(srcValue.Sign() != 0)
		return nil
	case reflect.Ptr:
		child := reflect.New(dstType.Elem())
		if err := AssignHexInt(child.Elem(), srcValue); err == nil {
			dstValue.Set(child)
			return nil
		}
	}
	return scoreresult.IllegalFormatError.Errorf("IncompatibleTypeForInt(type=%s)", dstType)
}

func AssignString(dstValue reflect.Value, srcValue string) error {
	dstType := dstValue.Type()
	if sliceOfByteType == dstType {
		dstValue.SetBytes([]byte(srcValue))
		return nil
	}
	dstKind := dstType.Kind()
	switch dstKind {
	case reflect.String:
		dstValue.SetString(srcValue)
		return nil
	case reflect.Ptr:
		child := reflect.New(dstType.Elem())
		if err := AssignString(child.Elem(), srcValue); err == nil {
			dstValue.Set(child)
			return nil
		}
	}
	return scoreresult.IllegalFormatError.Errorf("IncompatibleTypeForString(type=%s)", dstType)
}

func AssignBytes(dstValue reflect.Value, srcValue []byte) error {
	dstType := dstValue.Type()
	if sliceOfByteType == dstType {
		dstValue.SetBytes(srcValue)
		return nil
	}
	dstKind := dstType.Kind()
	switch dstKind {
	case reflect.String:
		dstValue.SetString(string(srcValue))
		return nil
	case reflect.Ptr:
		child := reflect.New(dstType.Elem())
		if err := AssignBytes(child.Elem(), srcValue); err == nil {
			dstValue.Set(child)
			return nil
		}
	}
	return scoreresult.IllegalFormatError.Errorf("IncompatibleTypeForBytes(type=%s)", dstType)
}

func AssignBool(dstValue reflect.Value, srcValue bool) error {
	dstType := dstValue.Type()
	dstKind := dstType.Kind()
	switch dstKind {
	case reflect.Bool:
		dstValue.SetBool(srcValue)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if srcValue {
			dstValue.SetUint(1)
		} else {
			dstValue.SetUint(0)
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if srcValue {
			dstValue.SetInt(1)
		} else {
			dstValue.SetInt(0)
		}
		return nil
	case reflect.Ptr:
		child := reflect.New(dstType.Elem())
		if err := AssignBool(child.Elem(), srcValue); err == nil {
			dstValue.Set(child)
			return nil
		}
	}
	return scoreresult.IllegalFormatError.Errorf("IncompatibleTypeForBytes(type=%s)", dstType)
}

func AssignAddress(dstValue reflect.Value, srcValue *common.Address) error {
	dstType := dstValue.Type()
	if ptrOfAddressType.AssignableTo(dstType) {
		dstValue.Set(reflect.ValueOf(srcValue))
		return nil
	}
	dstKind := dstType.Kind()
	switch dstKind {
	case reflect.Ptr:
		child := reflect.New(dstType.Elem())
		if err := AssignAddress(child.Elem(), srcValue); err == nil {
			dstValue.Set(child)
			return nil
		}
	}
	return scoreresult.IllegalFormatError.Errorf("IncompatibleTypeForAddress(type=%s)", dstType)
}

func AssignList(dstValue reflect.Value, srcValue []interface{}) error {
	dstType := dstValue.Type()
	dstKind := dstType.Kind()
	switch dstKind {
	case reflect.Slice:
		value := reflect.MakeSlice(dstType.Elem(), len(srcValue), len(srcValue))
		for i, v := range srcValue {
			child := value.Index(i)
			if err := AssignParameter(child, v); err != nil {
				return err
			}
		}
		dstValue.Set(value)
		return nil
	case reflect.Ptr:
		child := reflect.New(dstType.Elem())
		if err := AssignList(child.Elem(), srcValue); err == nil {
			dstValue.Set(child)
			return nil
		}
	}
	return scoreresult.IllegalFormatError.Errorf("IncompatibleTypeForList(type=%s)", dstType)
}

func AssignDict(dstValue reflect.Value, srcValue map[string]interface{}) error {
	dstType := dstValue.Type()
	dstKind := dstType.Kind()
typeHandler:
	switch dstKind {
	case reflect.Map:
		if dstType.Key().Kind() != reflect.String {
			break
		}
		value := reflect.MakeMap(dstType)
		for k, v := range srcValue {
			child := reflect.New(dstType.Elem()).Elem()
			if err := AssignParameter(child, v); err != nil {
				break typeHandler
			}
			value.SetMapIndex(reflect.ValueOf(k), child)
		}
		dstValue.Set(value)
		return nil
	case reflect.Struct:
		// TODO support struct
	case reflect.Ptr:
		child := reflect.New(dstType.Elem())
		if err := AssignDict(child.Elem(), srcValue); err == nil {
			dstValue.Set(child)
			return nil
		}
	}
	return scoreresult.IllegalFormatError.Errorf("IncompatibleTypeForList(type=%s)", dstType)
}

func AssignParameter(dstValue reflect.Value, value interface{}) error {
	// handle nil
	if value == nil {
		return nil
	}

	// general assignment
	srcValue := reflect.ValueOf(value)
	srcType := srcValue.Type()
	dstType := dstValue.Type()
	if srcType.AssignableTo(dstType) {
		dstValue.Set(srcValue)
		return nil
	}

	// let's handle conversion
	switch obj := value.(type) {
	case *common.HexInt:
		return AssignHexInt(dstValue, obj)
	case string:
		return AssignString(dstValue, obj)
	case map[string]interface{}:
		return AssignDict(dstValue, obj)
	case []interface{}:
		return AssignList(dstValue, obj)
	case []byte:
		return AssignBytes(dstValue, obj)
	case bool:
		return AssignBool(dstValue, obj)
	case *common.Address:
		return AssignAddress(dstValue, obj)
	}
	return scoreresult.InvalidParameterError.Errorf("UnknownInputType(%s)", srcType)
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
		return scoreresult.InvalidParameterError.Wrap(err, "IncompatibleParameter"),
			nil, steps
	} else {
		params = ps.([]interface{})
	}

	if len(params) != mType.NumIn() {
		return scoreresult.IllegalFormatError.Errorf(
			"NotEnoughParameter(exp=%d,real=%d)",
			mType.NumIn(), len(params)), nil, steps
	}

	objects := make([]reflect.Value, len(params))
	for i, p := range params {
		oType := mType.In(i)
		oValue := reflect.New(oType).Elem()
		if err := AssignParameter(oValue, p); err != nil {
			return errors.Wrapf(
					err,
					"InCompatibleType(to=%s,with=%T)",
					oType,
					p,
				),
				nil, steps
		}
		objects[i] = oValue
	}

	r := m.Call(objects)
	rLen := len(r)

	if rLen > 0 {
		last := r[rLen-1].Interface()
		if last != nil {
			if err, ok := last.(error); ok {
				status = err
			} else {
				status = scoreresult.UnknownFailureError.New("InvalidReturnValue")
			}
		} else if rLen >= 2 {
			if rLen == 2 {
				if ret, err := common.EncodeAny(r[0].Interface()); err != nil {
					status = scoreresult.UnknownFailureError.Wrap(err, "InvalidReturnValue")
				} else {
					result = ret
				}
			} else {
				panic("Not implemented")
			}
		}
	}
	return
}
