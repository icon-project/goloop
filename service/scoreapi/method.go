package scoreapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

type MethodType int

const (
	Function MethodType = iota
	Fallback
	Event
)

func (t MethodType) String() string {
	switch t {
	case Function:
		return "function"
	case Fallback:
		return "fallback"
	case Event:
		return "eventlog"
	default:
		log.Panicf("Fail to convert MethodType=%d", t)
		return "Unknown"
	}
}

type DataType int

const (
	Unknown DataType = iota
	Integer
	String
	Bytes
	Bool
	Address
	List
	Dict
)

func (t DataType) String() string {
	switch t {
	case Integer:
		return "int"
	case String:
		return "str"
	case Bytes:
		return "bytes"
	case Bool:
		return "bool"
	case Address:
		return "Address"
	case List:
		return "list"
	case Dict:
		return "dict"
	default:
		log.Panicf("Fail to convert DataType=%d", t)
		return "Unknown"
	}
}

func (t DataType) DecodeForJSON(bs []byte) interface{} {
	if bs == nil {
		return nil
	}
	switch t {
	case Integer:
		var i common.HexInt
		i.SetBytes(bs)
		return &i
	case String:
		return string(bs)
	case Bytes:
		return common.HexBytes(bs)
	case Bool:
		if (len(bs) == 1 && bs[0] == 0) || len(bs) == 0 {
			return "0x0"
		} else {
			return "0x1"
		}
	case Address:
		addr := new(common.Address)
		if err := addr.SetBytes(bs); err != nil {
			return nil
		}
		return addr
	default:
		log.Panicf("Unknown DataType=%d", t)
		return nil
	}
}

func (t DataType) Decode(bs []byte) interface{} {
	if bs == nil {
		return nil
	}
	switch t {
	case Integer:
		var i common.HexInt
		if len(bs) > 0 {
			i.SetBytes(bs)
		}
		return &i
	case String:
		return string(bs)
	case Bytes:
		return bs
	case Bool:
		if (len(bs) == 1 && bs[0] == 0) || len(bs) == 0 {
			return false
		} else {
			return true
		}
	case Address:
		addr := new(common.Address)
		if err := addr.SetBytes(bs); err != nil {
			return nil
		}
		return addr
	default:
		log.Panicf("Unknown DataType=%d", t)
		return nil
	}
}

func (t DataType) ValidateBytes(bs []byte) error {
	if bs == nil {
		return nil
	}
	switch t {
	case Integer:
		if len(bs) == 0 {
			return errors.IllegalArgumentError.New("InvalidIntegerBytes")
		}
	case Bool:
		if len(bs) != 1 {
			return errors.IllegalArgumentError.Errorf("InvalidBoolBytes(bs=<%#x>)", bs)
		}
		if bs[0] > 1 {
			return errors.IllegalArgumentError.Errorf("InvalidBoolBytes(bs=<%#x>)", bs)
		}
	case Address:
		var addr common.Address
		if err := addr.SetBytes(bs); err != nil {
			return errors.IllegalArgumentError.New("InvalidAddressBytes")
		}
	case String:
		if !utf8.Valid(bs) {
			return errors.IllegalArgumentError.New("InvalidUTF8Chars")
		}
	}
	return nil
}

var inputTypeTag = map[DataType]uint8{
	Integer: common.TypeInt,
	String:  codec.TypeString,
	Bytes:   codec.TypeBytes,
	Bool:    codec.TypeBool,
	Address: common.TypeAddress,
}

var outputTypeTag = map[DataType]struct {
	tag      uint8
	nullable bool
}{
	Integer: {common.TypeInt, false},
	String:  {codec.TypeString, false},
	Bytes:   {codec.TypeBytes, true},
	Bool:    {codec.TypeBool, false},
	Address: {common.TypeAddress, true},
	List:    {codec.TypeList, true},
	Dict:    {codec.TypeDict, true},
}

func (t DataType) ValidateInput(obj *codec.TypedObj, nullable bool) error {
	if typeTag, ok := inputTypeTag[t]; !ok {
		return errors.IllegalArgumentError.Errorf("UnknownType(%d)", t)
	} else {
		if typeTag == obj.Type {
			return nil
		}
		if obj.Type == codec.TypeNil && nullable {
			return nil
		}
		return errors.IllegalArgumentError.Errorf(
			"InvalidType(exp=%s,type=%d)", t, typeTag)
	}
}

func (t DataType) ValidateOutput(obj *codec.TypedObj) error {
	if obj == nil {
		obj = codec.Nil
	}
	if typeTag, ok := outputTypeTag[t]; !ok {
		return errors.IllegalArgumentError.Errorf("UnknownType(%d)", t)
	} else {
		if typeTag.tag == obj.Type {
			return nil
		}
		if obj.Type == codec.TypeNil && typeTag.nullable {
			return nil
		}
		return errors.IllegalArgumentError.Errorf(
			"InvalidType(exp=%d,type=%d)", typeTag.tag, obj.Type)
	}
}

// DataTypeOf returns type for the specified name.
func DataTypeOf(s string) DataType {
	switch s {
	case "bool":
		return Bool
	case "int":
		return Integer
	case "str":
		return String
	case "bytes":
		return Bytes
	case "Address":
		return Address
	case "list":
		return List
	case "dict":
		return Dict
	default:
		return Unknown
	}
}

const (
	FlagReadOnly = 1 << iota
	FlagExternal
	FlagPayable
	FlagIsolated
)

type Parameter struct {
	Name    string
	Type    DataType
	Default []byte
}

type Method struct {
	Type    MethodType
	Name    string
	Flags   int
	Indexed int
	Inputs  []Parameter
	Outputs []DataType
}

func (a *Method) IsPayable() bool {
	return a.Type != Event && (a.Flags&FlagPayable) != 0
}

func (a *Method) IsReadOnly() bool {
	return a.Type == Function && (a.Flags&FlagReadOnly) != 0
}

func (a *Method) IsExternal() bool {
	return a.Type == Function && (a.Flags&(FlagExternal|FlagReadOnly)) != 0
}

func (a *Method) IsIsolated() bool {
	return a.Type != Event && (a.Flags&FlagIsolated) != 0
}

func (a *Method) IsCallable() bool {
	return a.Type != Event
}

func (a *Method) IsFallback() bool {
	return a.Type == Fallback
}

func (a *Method) IsEvent() bool {
	return a.Type == Event
}
func (a *Method) ToJSON(version module.JSONVersion) (interface{}, error) {
	m := make(map[string]interface{})
	m["type"] = a.Type.String()
	m["name"] = a.Name

	inputs := make([]interface{}, len(a.Inputs))
	for i, input := range a.Inputs {
		io := make(map[string]interface{})
		io["name"] = input.Name
		io["type"] = input.Type.String()
		if a.Type == Event {
			if i < a.Indexed {
				io["indexed"] = "0x1"
			}
		} else {
			if i >= a.Indexed {
				io["default"] = input.Type.DecodeForJSON(input.Default)
			}
		}
		inputs[i] = io
	}
	m["inputs"] = inputs

	outputs := make([]interface{}, len(a.Outputs))
	for i, output := range a.Outputs {
		oo := make(map[string]interface{})
		oo["type"] = output.String()
		outputs[i] = oo
	}
	m["outputs"] = outputs
	if (a.Flags & FlagReadOnly) != 0 {
		m["readonly"] = "0x1"
	}
	if (a.Flags & FlagPayable) != 0 {
		m["payable"] = "0x1"
	}
	if (a.Flags & FlagIsolated) != 0 {
		m["isolated"] = "0x1"
	}
	return m, nil
}

func (a *Method) EnsureParamsSequential(paramObj *codec.TypedObj) (*codec.TypedObj, error) {
	if paramObj.Type == codec.TypeList {
		tol := paramObj.Object.([]*codec.TypedObj)
		if len(tol) < a.Indexed {
			return nil, scoreresult.InvalidParameterError.Errorf(
				"NotEnoughParameters(given=%d,required=%d)", len(tol), a.Indexed)
		}
		if len(tol) > len(a.Inputs) {
			return nil, scoreresult.InvalidParameterError.Errorf(
				"TooManyParameters(given=%d,all=%d)", len(tol), len(a.Inputs))
		}
		tolNew := tol
		for i, input := range a.Inputs {
			inputType := a.Inputs[i].Type
			if i < len(tol) {
				to := tol[i]
				nullable := (i >= a.Indexed) && input.Default == nil
				if err := inputType.ValidateInput(to, nullable); err != nil {
					return nil, err
				}
			} else {
				tolNew = append(tolNew,
					common.MustEncodeAny(inputType.Decode(input.Default)))
			}
		}
		paramObj.Object = tolNew
		return paramObj, nil
	}

	if paramObj.Type != codec.TypeDict {
		return nil, scoreresult.ErrInvalidParameter
	}
	params, ok := paramObj.Object.(map[string]*codec.TypedObj)
	if !ok {
		return nil, scoreresult.InvalidParameterError.Errorf(
			"FailToCastDictToMap(type=%[1]T, obj=%+[1]v)", paramObj.Object)
	}
	inputs := make([]interface{}, len(a.Inputs))
	for i, input := range a.Inputs {
		if obj, ok := params[input.Name]; ok {
			nullable := (i >= a.Indexed) && input.Default == nil
			if err := input.Type.ValidateInput(obj, nullable); err != nil {
				return nil, scoreresult.InvalidParameterError.Wrapf(err,
					"InvalidParameter(exp=%s, value=%T)", input.Type, obj)
			}
			inputs[i] = obj
		} else {
			if i >= a.Indexed {
				inputs[i] = input.Type.Decode(input.Default)
			} else {
				return nil, scoreresult.InvalidParameterError.Errorf(
					"MissingParameter(name=%s)", input.Name)
			}
		}
	}
	return common.MustEncodeAny(inputs), nil
}

func (a *Method) Signature() string {
	args := make([]string, len(a.Inputs))
	for i := 0; i < len(args); i++ {
		args[i] = a.Inputs[i].Type.String()
	}
	return fmt.Sprintf("%s(%s)", a.Name, strings.Join(args, ","))
}

func (a *Method) CheckEventData(indexed [][]byte, data [][]byte) error {
	if len(indexed)+len(data) != len(a.Inputs)+1 {
		return IllegalEventError.Errorf(
			"InvalidEventData(exp=%d,given=%d)",
			len(a.Inputs)+1, len(indexed)+len(data))
	}
	if len(indexed) != a.Indexed+1 {
		return IllegalEventError.Errorf(
			"InvalidIndexCount(exp=%d,given=%d)", a.Indexed, len(indexed)-1)
	}
	for i, p := range a.Inputs {
		var input []byte
		if i < len(indexed)-1 {
			input = indexed[i+1]
		} else {
			input = data[i+1-len(indexed)]
		}
		if err := p.Type.ValidateBytes(input); err != nil {
			return IllegalEventError.Wrapf(err,
				"IllegalEvent(sig=%s,idx=%d,data=0x%#x)",
				a.Signature(), i, input)
		}
	}
	return nil
}

func (a *Method) ConvertParamsToTypedObj(bs []byte) (*codec.TypedObj, error) {
	var params map[string]string
	if len(bs) > 0 {
		if err := json.Unmarshal(bs, &params); err != nil {
			return nil, scoreresult.WithStatus(err, module.StatusInvalidParameter)
		}
	}
	matched := 0
	inputs := make([]interface{}, len(a.Inputs))
	for i, input := range a.Inputs {
		param, ok := params[input.Name]
		if !ok {
			if i >= a.Indexed {
				inputs[i] = input.Type.Decode(input.Default)
				continue
			}
			return nil, scoreresult.Errorf(module.StatusInvalidParameter,
				"MissingParam(param=%s)", input.Name)
		}

		matched += 1

		switch input.Type {
		case Integer:
			var value common.HexInt
			if _, ok := value.SetString(param, 0); !ok {
				return nil, scoreresult.Errorf(module.StatusInvalidParameter,
					"FailToConvertInteger(param=%s,value=%s)", input.Name, param)
			}
			inputs[i] = &value
		case String:
			inputs[i] = param
		case Bytes:
			if len(param) < 2 || param[0:2] != "0x" {
				return nil, scoreresult.Errorf(module.StatusInvalidParameter,
					"InvalidParam(param=%s)", param)
			}
			value, err := hex.DecodeString(param[2:])
			if err != nil {
				return nil, scoreresult.WithStatus(err, module.StatusInvalidParameter)
			}
			inputs[i] = value
		case Bool:
			switch param {
			case "0x1":
				inputs[i] = true
			case "0x0":
				inputs[i] = false
			default:
				return nil, scoreresult.Errorf(module.StatusInvalidParameter,
					"IllegalParamForBool(param=%s)", param)
			}
		case Address:
			var value common.Address
			if err := value.SetString(param); err != nil {
				return nil, scoreresult.WithStatus(err, module.StatusInvalidParameter)
			}
			inputs[i] = &value
		default:
			return nil, scoreresult.Errorf(module.StatusInvalidParameter,
				"UnknownType(%d)", input.Type)
		}
	}

	if matched != len(params) {
		return nil, scoreresult.Errorf(module.StatusInvalidParameter,
			"UnexpectedParam(%v)\n", params)
	}

	if to, err := common.EncodeAny(inputs); err != nil {
		return nil, scoreresult.WithStatus(err, module.StatusInvalidParameter)
	} else {
		return to, nil
	}
}

func (a *Method) EnsureResult(result *codec.TypedObj) error {
	if a == nil {
		return scoreresult.MethodNotFoundError.New("NoMethod")
	}
	if result == nil {
		result = codec.Nil
	}
	if len(a.Outputs) == 0 {
		if result.Type == codec.TypeNil {
			return nil
		}
		if !a.IsReadOnly() {
			// Some of execution environment returns empty
			// outputs for writable functions with outputs.
			// To support old versions, it ignores
			// empty outputs.
			return nil
		}
		return scoreresult.UnknownFailureError.Errorf(
			"InvalidReturn(exp=None,real=%d)", result.Type)
	}
	var results []*codec.TypedObj
	if len(a.Outputs) == 1 {
		results = []*codec.TypedObj{result}
	} else {
		if result.Type != codec.TypeList {
			return scoreresult.UnknownFailureError.Errorf(
				"InvalidReturnType(type=%d)", result.Type)
		}
		if rs, ok := result.Object.([]*codec.TypedObj); !ok {
			return scoreresult.UnknownFailureError.Errorf(
				"InvalidReturnType(type=%T)", result.Object)
		} else {
			results = rs
		}
	}
	if len(a.Outputs) != len(results) {
		return scoreresult.UnknownFailureError.Errorf(
			"InvalidReturnLength(exp=%d,real=%d)",
			len(a.Outputs), len(results))
	}
	for i, o := range results {
		if err := a.Outputs[i].ValidateOutput(o); err != nil {
			return scoreresult.UnknownFailureError.Wrapf(err,
				"InvalidReturnType(idx=%d)", i)
		}
	}
	return nil
}
