package scoreapi

import (
	"bytes"
	"encoding/json"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	FallbackMethodName = ""
)

type Info struct {
	methods   []*Method
	methodMap map[string]*Method
}

func (info *Info) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(info.methods)
}

func (info *Info) RLPDecodeSelf(d codec.Decoder) error {
	if err := d.Decode(&info.methods); err != nil {
		return err
	}
	if info.methods == nil {
		return codec.ErrNilValue
	}
	info.buildMethodMap()
	return nil
}

func (info *Info) Bytes() ([]byte, error) {
	return codec.MarshalToBytes(info.methods)
}

func (info *Info) buildMethodMap() {
	m := make(map[string]*Method)
	for _, method := range info.methods {
		if method.IsEvent() {
			m[method.Signature()] = method
		} else {
			if method.IsFallback() {
				m[FallbackMethodName] = method
			} else if method.Name != FallbackMethodName {
				m[method.Name] = method
			}
		}
	}
	info.methodMap = m
}

func (info *Info) GetMethod(name string) *Method {
	if method, ok := info.methodMap[name]; ok {
		return method
	} else {
		return nil
	}
}

func (info *Info) EnsureParamsSequential(method string, params *codec.TypedObj) (*codec.TypedObj, error) {
	m := info.GetMethod(method)
	if m == nil {
		return nil, scoreresult.ErrMethodNotFound
	}
	return m.EnsureParamsSequential(params)
}

func (info *Info) ConvertParamsToTypedObj(method string, params []byte) (*codec.TypedObj, error) {
	m := info.GetMethod(method)
	if m == nil {
		return nil, scoreresult.ErrMethodNotFound
	}
	return m.ConvertParamsToTypedObj(params, false)
}

func (info *Info) CheckEventData(indexed [][]byte, data [][]byte) error {
	if len(indexed) < 1 {
		return ErrNoSignature
	}
	s := string(indexed[0])
	m := info.GetMethod(s)
	if m == nil {
		return errors.ErrNotFound
	}
	return m.CheckEventData(indexed, data)
}

func (info *Info) ToJSON(v module.JSONVersion) (interface{}, error) {
	jso := make([]interface{}, 0, len(info.methods))
	for _, method := range info.methods {
		if !method.IsExternal() && !method.IsEvent() && !method.IsFallback() {
			continue
		}
		if json, err := method.ToJSON(v); err != nil {
			return nil, err
		} else {
			jso = append(jso, json)
		}
	}
	return jso, nil
}

func (info *Info) String() string {
	if info == nil {
		return "nil"
	}
	jso, _ := info.ToJSON(3)
	bs, _ := json.Marshal(jso)
	return string(bs)
}

func (info *Info) Equal(info2 *Info) bool {
	if info == info2 {
		return true
	}
	if info == nil || info2 == nil {
		return false
	}
	bs1, _ := info.Bytes()
	bs2, _ := info2.Bytes()
	return bytes.Equal(bs1, bs2)
}

func NewInfo(methods []*Method) *Info {
	info := &Info{
		methods: methods,
	}
	info.buildMethodMap()
	return info
}
