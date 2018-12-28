package scoreapi

import (
	"encoding/hex"
	"encoding/json"
	"log"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/pkg/errors"
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
		return "event"
	default:
		log.Panicf("Fail to convert MethodType=%d", t)
		return "Unknown"
	}
}

type DataType int

const (
	Integer DataType = iota + 1
	String
	Bytes
	Bool
	Address
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
	default:
		log.Panicf("Fail to convert DataType=%d", t)
		return "Unknown"
	}
}

const (
	FlagReadOnly = 1 << iota
	FlagExternal
	FlagPayable
)

type Parameter struct {
	Name string
	Type DataType
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
	return a.Type == Function && (a.Flags&FlagExternal) != 0
}

func (a *Method) IsCallable() bool {
	return a.Type != Event
}

func (a *Method) ToJSON(version int) (interface{}, error) {
	m := make(map[string]interface{})
	m["type"] = a.Type.String()
	m["name"] = a.Name

	inputs := make([]interface{}, len(a.Inputs))
	for i, input := range a.Inputs {
		io := make(map[string]interface{})
		io["name"] = input.Name
		io["type"] = input.Type.String()
		if i < a.Indexed {
			io["indexed"] = "0x1"
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
	return m, nil
}

func (a *Method) ConvertParamsToTypedObj(bs []byte) (*codec.TypedObj, error) {
	var params map[string]string
	if err := json.Unmarshal(bs, &params); err != nil {
		return nil, err
	}
	inputs := make([]interface{}, len(a.Inputs))
	for i, input := range a.Inputs {
		param, ok := params[input.Name]
		if !ok {
			return nil, errors.Errorf("MissingParam(param=%s)", input.Name)
		}
		switch input.Type {
		case Integer:
			var value common.HexInt
			if _, ok := value.SetString(param, 0); !ok {
				return nil, errors.Errorf("FailToConvertInteger(param=%s,value=%s)", input.Name, param)
			}
			inputs[i] = &value
		case String:
			inputs[i] = param
		case Bytes:
			if param[0:2] != "0x" {
				return nil, errors.Errorf("InvalidPrefix(prefix=%s)", param[0:2])
			}
			value, err := hex.DecodeString(param[2:])
			if err != nil {
				return nil, err
			}
			inputs[i] = value
		case Bool:
			switch param {
			case "0x1":
				inputs[i] = true
			case "0x0":
				inputs[i] = false
			default:
				return nil, errors.Errorf("IllegalParamForBool(param=%s)", param)
			}
		case Address:
			var value common.Address
			if err := value.SetString(param); err != nil {
				return nil, err
			}
			inputs[i] = &value
		default:
			return nil, errors.Errorf("UnknownType(%d)", input.Type)
		}
	}
	return common.EncodeAny(inputs)
}
