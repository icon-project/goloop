package scoreapi

import (
	"encoding/json"

	"github.com/icon-project/goloop/common/codec"
	"github.com/pkg/errors"
	ugorji "github.com/ugorji/go/codec"
)

type Info struct {
	methods   []*Method
	methodMap map[string]*Method
}

func (info *Info) CodecEncodeSelf(e *ugorji.Encoder) {
	e.MustEncode(info.methods)
}

func (info *Info) CodecDecodeSelf(d *ugorji.Decoder) {
	d.MustDecode(&info.methods)
	info.buildMethodMap()
}

func (info *Info) Bytes() ([]byte, error) {
	return codec.MarshalToBytes(info.methods)
}

func (info *Info) buildMethodMap() {
	m := make(map[string]*Method)
	for _, method := range info.methods {
		m[method.Name] = method
	}
	info.methodMap = m
}

func (info *Info) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &info.methods)
	if err != nil {
		info.buildMethodMap()
	}
	return err
}

func (info *Info) GetMethod(name string) *Method {
	if method, ok := info.methodMap[name]; ok {
		return method
	} else {
		return nil
	}
}

func (info *Info) ConvertParamsToTypedObj(method string, params []byte) (*codec.TypedObj, error) {
	m := info.GetMethod(method)
	if m == nil {
		return nil, errors.Errorf("NoMethod(name=%s)", method)
	}
	return m.ConvertParamsToTypedObj(params)
}

func (info *Info) ToJSON(v int) (interface{}, error) {
	jso := make([]interface{}, len(info.methods))
	for i, method := range info.methods {
		var err error
		jso[i], err = method.ToJSON(v)
		if err != nil {
			return nil, err
		}
	}
	return jso, nil
}

func (info *Info) String() string {
	jso, _ := info.ToJSON(3)
	bs, _ := json.Marshal(jso)
	return string(bs)
}

func NewInfo(methods []*Method) *Info {
	info := &Info{
		methods: methods,
	}
	info.buildMethodMap()
	return info
}
