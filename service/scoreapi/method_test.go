package scoreapi

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

func TestMethod_EnsureResult(t *testing.T) {
	type fields struct {
		Outputs []DataType
		Flags   int
	}
	type args struct {
		result *codec.TypedObj
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"NoReturnWithNilPtr",
			fields{[]DataType{}, FlagExternal},
			args{nil},
			false,
		},
		{
			"NoReturnWithNilObj",
			fields{[]DataType{}, FlagExternal},
			args{codec.Nil},
			false,
		},
		{
			"NoReturnWithInt",
			fields{[]DataType{}, FlagExternal},
			args{common.MustEncodeAny(1)},
			false,
		},
		{
			"NoReturnWithIntString",
			fields{[]DataType{}, FlagExternal},
			args{common.MustEncodeAny([]interface{}{1, "string"})},
			false,
		},
		{
			"NoReturnWithNilPtrRDOnly",
			fields{[]DataType{}, FlagExternal | FlagReadOnly},
			args{nil},
			false,
		},
		{
			"NoReturnWithNilObjRDOnly",
			fields{[]DataType{}, FlagExternal | FlagReadOnly},
			args{codec.Nil},
			false,
		},
		{
			"NoReturnWithIntRDOnly",
			fields{[]DataType{}, FlagExternal | FlagReadOnly},
			args{common.MustEncodeAny(1)},
			true,
		},
		{
			"NoReturnWithIntStringRDOnly",
			fields{[]DataType{}, FlagExternal | FlagReadOnly},
			args{common.MustEncodeAny([]interface{}{1, "string"})},
			true,
		},
		{
			"IntReturnWithNilPtr",
			fields{[]DataType{Integer}, FlagExternal},
			args{nil},
			true,
		},
		{
			"IntReturnWithNilObj",
			fields{[]DataType{Integer}, FlagExternal},
			args{codec.Nil},
			true,
		},
		{
			"IntReturnWithStringObj",
			fields{[]DataType{Integer}, FlagExternal},
			args{common.MustEncodeAny("test")},
			true,
		},
		{
			"IntReturnWithIntString",
			fields{[]DataType{Integer}, FlagExternal},
			args{common.MustEncodeAny([]interface{}{0, "test"})},
			true,
		},
		{
			"IntStringReturnWithNil",
			fields{[]DataType{Integer, String}, FlagExternal},
			args{codec.Nil},
			true,
		},
		{
			"IntStringReturnWithInt",
			fields{[]DataType{Integer, String}, FlagExternal},
			args{common.MustEncodeAny([]interface{}{0})},
			true,
		},
		{
			"IntStringReturnWithIntString",
			fields{[]DataType{Integer, String}, FlagExternal},
			args{common.MustEncodeAny([]interface{}{0, "string"})},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Method{
				Type:    Function,
				Name:    "test",
				Flags:   tt.fields.Flags,
				Indexed: 0,
				Inputs:  []Parameter{},
				Outputs: tt.fields.Outputs,
			}
			if err := a.EnsureResult(tt.args.result); (err != nil) != tt.wantErr {
				t.Errorf("EnsureResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMethod_ConvertParamsToTypedObj(t *testing.T) {
	type fields struct {
		Type    MethodType
		Name    string
		Flags   int
		Indexed int
		Inputs  []Parameter
		Outputs []DataType
	}
	type args struct {
		bs []byte
	}
	var tests = []struct {
		name    string
		fields  fields
		args    args
		want    *codec.TypedObj
		wantErr bool
	}{
		{
			name: "DictParams",
			fields: fields{
				Type:    Function,
				Name:    "transfer",
				Flags:   FlagExternal,
				Indexed: 2,
				Inputs: []Parameter{
					{
						Name: "_to",
						Type: Address,
					},
					{
						Name: "_value",
						Type: Integer,
					},
					{
						Name: "_data",
						Type: Bytes,
					},
				},
			},
			args: args{[]byte("{\"_to\":\"hxff9221db215ce1a511cbe0a12ff9eb70be4e5764\",\"_value\":\"0xa\"}")},
			want: common.MustEncodeAny([]interface{}{
				common.MustNewAddressFromString("hxff9221db215ce1a511cbe0a12ff9eb70be4e5764"),
				common.NewHexInt(10),
				nil,
			}),
			wantErr: false,
		},
		{
			name: "ListParams",
			fields: fields{
				Type:    Function,
				Name:    "transfer",
				Flags:   FlagExternal,
				Indexed: 2,
				Inputs: []Parameter{
					{
						Name: "_to",
						Type: Address,
					},
					{
						Name: "_value",
						Type: Integer,
					},
					{
						Name: "_data",
						Type: Bytes,
					},
				},
			},
			args: args{[]byte("[\"hxff9221db215ce1a511cbe0a12ff9eb70be4e5764\",\"0xa\"]")},
			want: common.MustEncodeAny([]interface{}{
				common.MustNewAddressFromString("hxff9221db215ce1a511cbe0a12ff9eb70be4e5764"),
				common.NewHexInt(10),
				nil,
			}),
			wantErr: false,
		},
		{
			name: "NilParams",
			fields: fields{
				Type:    Function,
				Name:    "check",
				Flags:   FlagExternal,
				Indexed: 1,
				Inputs: []Parameter{
					{
						Name: "_data",
						Type: Bytes,
					},
				},
			},
			args:    args{nil},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoParams",
			fields: fields{
				Type:    Function,
				Name:    "methodWithNoParams",
				Flags:   FlagExternal,
				Indexed: 0,
			},
			args:    args{nil},
			want:    common.MustEncodeAny([]interface{}{}),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Method{
				Type:    tt.fields.Type,
				Name:    tt.fields.Name,
				Flags:   tt.fields.Flags,
				Indexed: tt.fields.Indexed,
				Inputs:  tt.fields.Inputs,
				Outputs: tt.fields.Outputs,
			}
			got, err := a.ConvertParamsToTypedObj(tt.args.bs, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertParamsToTypedObj() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertParamsToTypedObj() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMethod(t *testing.T) {
	var tests = []struct {
		name   string
		method *Method
		sig    string
		jsoErr bool
		jso    interface{}
	}{
		{
			"SimpleVoid",
			&Method{
				Type:  Function,
				Name:  "test",
				Flags: FlagExternal,
			},
			"test()",
			false,
			map[string]interface{}{
				"type":    "function",
				"name":    "test",
				"inputs":  []interface{}{},
				"outputs": []interface{}{},
			},
		},
		{
			"BasicFunction",
			&Method{
				Type:    Function,
				Name:    "test",
				Flags:   FlagExternal | FlagPayable,
				Indexed: 3,
				Inputs: []Parameter{
					{
						Name: "_idx",
						Type: Integer,
					},
					{
						Name: "_name",
						Type: String,
					},
					{
						Name: "_addr",
						Type: Address,
					},
					{
						Name: "_data",
						Type: Bytes,
					},
				},
				Outputs: []DataType{Dict},
			},
			"test(int,str,Address,bytes)",
			false,
			map[string]interface{}{
				"type": "function",
				"name": "test",
				"inputs": []interface{}{
					map[string]interface{}{
						"name": "_idx",
						"type": "int",
					},
					map[string]interface{}{
						"name": "_name",
						"type": "str",
					},
					map[string]interface{}{
						"name": "_addr",
						"type": "Address",
					},
					map[string]interface{}{
						"name":    "_data",
						"type":    "bytes",
						"default": nil,
					},
				},
				"outputs": []interface{}{
					map[string]interface{}{
						"type": "dict",
					},
				},
				"payable": "0x1",
			},
		},
		{
			"ComplexFunction",
			&Method{
				Type:    Function,
				Name:    "complex_method",
				Flags:   FlagExternal,
				Indexed: 2,
				Inputs: []Parameter{
					{
						Name: "_addrs",
						Type: ListTypeOf(1, Address),
					},
					{
						Name: "_info",
						Type: ListTypeOf(1, Struct),
						Fields: []Field{
							{
								Name: "name",
								Type: String,
							},
							{
								Name: "delegation",
								Type: Integer,
							},
							{
								Name: "host",
								Type: Struct,
								Fields: []Field{
									{
										Name: "name",
										Type: String,
									},
									{
										Name: "url",
										Type: String,
									},
								},
							},
						},
					},
				},
				Outputs: []DataType{},
			},
			"complex_method([]Address,[]struct)",
			false,
			map[string]interface{}{
				"type": "function",
				"name": "complex_method",
				"inputs": []interface{}{
					map[string]interface{}{
						"name": "_addrs",
						"type": "[]Address",
					},
					map[string]interface{}{
						"name": "_info",
						"type": "[]struct",
						"fields": []interface{}{
							map[string]interface{}{
								"name": "name",
								"type": "str",
							},
							map[string]interface{}{
								"name": "delegation",
								"type": "int",
							},
							map[string]interface{}{
								"name": "host",
								"type": "struct",
								"fields": []interface{}{
									map[string]interface{}{
										"name": "name",
										"type": "str",
									},
									map[string]interface{}{
										"name": "url",
										"type": "str",
									},
								},
							},
						},
					},
				},
				"outputs": []interface{}{},
			},
		},
		{
			"Event1",
			&Method{
				Type:    Event,
				Name:    "MyEvent",
				Indexed: 2,
				Inputs: []Parameter{
					{
						Name: "addr",
						Type: Address,
					},
					{
						Name: "balance",
						Type: Integer,
					},
					{
						Name: "reason",
						Type: String,
					},
				},
			},
			"MyEvent(Address,int,str)",
			false,
			map[string]interface{}{
				"name": "MyEvent",
				"type": "eventlog",
				"inputs": []interface{}{
					map[string]interface{}{
						"name":    "addr",
						"type":    "Address",
						"indexed": "0x1",
					},
					map[string]interface{}{
						"name":    "balance",
						"type":    "int",
						"indexed": "0x1",
					},
					map[string]interface{}{
						"name": "reason",
						"type": "str",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// signature value test
			assert.Equal(t, tt.sig, tt.method.Signature())

			// json encoding test
			js1, err := tt.method.ToJSON(module.JSONVersionLast)
			if tt.jsoErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if err != nil {
				return
			}
			bs1, err := json.Marshal(js1)
			assert.NoError(t, err)
			bs0, err := json.Marshal(tt.jso)
			assert.NoError(t, err)
			assert.Equal(t, string(bs0), string(bs1))

			// bytes encode/decode test
			bs, err := codec.BC.MarshalToBytes(tt.method)
			assert.NoError(t, err)

			var m2 *Method
			remain, err := codec.BC.UnmarshalFromBytes(bs, &m2)
			assert.NoError(t, err)
			assert.Empty(t, remain)

			assert.Equal(t, tt.method, m2)
		})
	}
}

func TestDataListOf(t *testing.T) {
	dt := ListTypeOf(2, Integer)
	assert.True(t, dt.IsList())
	assert.Equal(t, 2, dt.ListDepth())

	dt1 := dt.Elem()
	assert.True(t, dt1.IsList())
	assert.Equal(t, 1, dt1.ListDepth())

	dt2 := dt1.Elem()
	assert.False(t, dt2.IsList())
	assert.Equal(t, 0, dt2.ListDepth())
	assert.Equal(t, Integer, dt2)
}

func TestDataTypeOf(t *testing.T) {
	var tests = []struct {
		name      string
		dtString  string
		dtValue   DataType
		listDepth int
	}{
		{
			name:     "int",
			dtString: "int",
			dtValue:  Integer,
		},
		{
			name:      "listOfInt",
			dtString:  "[][]int",
			dtValue:   ListTypeOf(2, Integer),
			listDepth: 2,
		},
		{
			name:     "invalid1",
			dtString: "integer",
			dtValue:  Unknown,
		},
		{
			name:     "invalid2",
			dtString: "[]]int",
			dtValue:  Unknown,
		},
		{
			name:      "listOfStruct",
			dtString:  "[]struct",
			dtValue:   ListTypeOf(1, Struct),
			listDepth: 1,
		},
		{
			name:     "bool",
			dtString: "bool",
			dtValue:  Bool,
		},
		{
			name:     "str",
			dtString: "str",
			dtValue:  String,
		},
		{
			name:     "address",
			dtString: "Address",
			dtValue:  Address,
		},
		{
			name:     "bytes",
			dtString: "bytes",
			dtValue:  Bytes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := DataTypeOf(tt.dtString)
			assert.Equal(t, tt.dtValue, dt)
			if dt == Unknown {
				return
			}
			assert.Equal(t, tt.listDepth, dt.ListDepth())
			assert.Equal(t, tt.listDepth > 0, dt.IsList())
		})
	}
}

func TestConvertBytesToTypedObj(t *testing.T) {
	var ErrExpectingError = errors.New("expected error")
	var ListOfInteger = ListTypeOf(1, Integer)
	var cases = []struct {
		name   string
		bytes  []byte
		values map[DataType]interface{}
	}{
		{
			name:  "nil",
			bytes: nil,
			values: map[DataType]interface{}{
				Unknown:       ErrExpectingError,
				String:        codec.Nil,
				Integer:       codec.Nil,
				Address:       codec.Nil,
				Bytes:         codec.Nil,
				Bool:          codec.Nil,
				ListOfInteger: codec.Nil,
				Struct:        codec.Nil,
			},
		},
		{
			name:  "0x2334",
			bytes: []byte("\x23\x34"),
			values: map[DataType]interface{}{
				Unknown:       ErrExpectingError,
				String:        common.MustEncodeAny("\x23\x34"),
				Integer:       common.MustEncodeAny(0x2334),
				Address:       ErrExpectingError,
				Bytes:         common.MustEncodeAny([]byte{0x23, 0x34}),
				Bool:          common.MustEncodeAny(true),
				ListOfInteger: ErrExpectingError,
				Struct:        ErrExpectingError,
			},
		},
		{
			name:  "0x01",
			bytes: []byte("\x01"),
			values: map[DataType]interface{}{
				String:  common.MustEncodeAny("\x01"),
				Integer: common.MustEncodeAny(0x1),
				Address: ErrExpectingError,
				Bytes:   common.MustEncodeAny([]byte{0x01}),
				Bool:    common.MustEncodeAny(true),
			},
		},
		{
			name:  "0x00",
			bytes: []byte("\x00"),
			values: map[DataType]interface{}{
				String:  common.MustEncodeAny("\x00"),
				Integer: common.MustEncodeAny(0x0),
				Address: ErrExpectingError,
				Bytes:   common.MustEncodeAny([]byte{0x00}),
				Bool:    common.MustEncodeAny(false),
			},
		},
		{
			name:  "0xf300",
			bytes: []byte("\xf3\x00"),
			values: map[DataType]interface{}{
				String:  common.MustEncodeAny("\xf3\x00"),
				Integer: common.MustEncodeAny(-3328),
				Address: ErrExpectingError,
				Bytes:   common.MustEncodeAny([]byte{0xf3, 0x00}),
				Bool:    common.MustEncodeAny(true),
			},
		},
		{
			name:  "0x012aa9b28a657e3121b75d3d4fe65e569398645d56",
			bytes: []byte("\x01\x2a\xa9\xb2\x8a\x65\x7e\x31\x21\xb7\x5d\x3d\x4f\xe6\x5e\x56\x93\x98\x64\x5d\x56"),
			values: map[DataType]interface{}{
				String:  common.MustEncodeAny("\x01\x2a\xa9\xb2\x8a\x65\x7e\x31\x21\xb7\x5d\x3d\x4f\xe6\x5e\x56\x93\x98\x64\x5d\x56"),
				Address: common.MustEncodeAny(common.MustNewAddressFromString("cx2aa9b28a657e3121b75d3d4fe65e569398645d56")),
				Bytes:   common.MustEncodeAny([]byte("\x01\x2a\xa9\xb2\x8a\x65\x7e\x31\x21\xb7\x5d\x3d\x4f\xe6\x5e\x56\x93\x98\x64\x5d\x56")),
				Bool:    common.MustEncodeAny(true),
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for dt, expect := range tt.values {
				t.Run(dt.String(), func(t *testing.T) {
					obj1, err := dt.ConvertBytesToTypedObj(tt.bytes)
					if expect == ErrExpectingError {
						assert.Error(t, err)
						return
					} else {
						assert.NoError(t, err)
					}
					assert.EqualValues(t, expect, obj1)
				})
			}
		})
	}
}

func TestEnsureParamsSequential(t *testing.T) {
	var method1 = &Method{
		Type:    Function,
		Name:    "transfer",
		Flags:   FlagExternal,
		Indexed: 2,
		Inputs: []Parameter{
			{
				Name: "_to",
				Type: Address,
			},
			{
				Name: "_amount",
				Type: Integer,
			},
			{
				Name: "_data",
				Type: Bytes,
			},
		},
	}
	var method2 = &Method{
		Type:    Function,
		Name:    "transfer",
		Flags:   FlagExternal,
		Indexed: 0,
		Inputs: []Parameter{
			{
				Name:    "_to",
				Type:    Address,
				Default: []byte{0x00, 0x12},
			},
		},
	}
	var cases = []struct {
		name    string
		method  *Method
		params  interface{}
		wantErr bool
		want    interface{}
	}{
		{
			name:   "FullParams",
			method: method1,
			params: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				1238,
				make([]byte, 0),
			},
			want: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				1238,
				make([]byte, 0),
			},
		},
		{
			name:   "PartialParams",
			method: method1,
			params: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				1238,
			},
			want: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				1238,
				nil,
			},
		},
		{
			name:   "MissingParams1",
			method: method1,
			params: []interface{}{
				common.MustNewAddressFromString("hx1234"),
			},
			wantErr: true,
		},
		{
			name:   "DictionaryFullParams",
			method: method1,
			params: map[string]interface{}{
				"_to":     common.MustNewAddressFromString("hx1234"),
				"_amount": 1566,
				"_data":   []byte{0x12, 0x34},
			},
			want: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				1566,
				[]byte{0x12, 0x34},
			},
		},
		{
			name:   "DictionaryPartialParams",
			method: method1,
			params: map[string]interface{}{
				"_to":     common.MustNewAddressFromString("hx1234"),
				"_amount": 1566,
			},
			want: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				1566,
				nil,
			},
		},
		{
			name:   "DictionaryMissingParams",
			method: method1,
			params: map[string]interface{}{
				"_to": common.MustNewAddressFromString("hx1234"),
			},
			wantErr: true,
		},
		{
			name:    "InvalidParam",
			method:  method1,
			params:  1234,
			wantErr: true,
		},
		{
			name:   "LargeParams",
			method: method1,
			params: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				1566,
				[]byte{0x12, 0x34},
				"dummy",
			},
			wantErr: true,
		},
		{
			name:   "NilParam",
			method: method1,
			params: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				nil,
				[]byte{0x12, 0x34},
			},
			wantErr: true,
		},
		{
			name:    "InvalidDefault1",
			method:  method2,
			params:  []interface{}{},
			wantErr: true,
		},
		{
			name:    "InvalidDefault2",
			method:  method2,
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name:   "InvalidParam",
			method: method1,
			params: map[string]interface{}{
				"_to":     common.MustNewAddressFromString("hx123456"),
				"_amount": nil,
			},
			wantErr: true,
		},
		{
			name:   "MixtureOfPositionAndDict",
			method: method1,
			params: map[string]interface{}{
				KeyForPositionalParameters: []interface{}{
					common.MustNewAddressFromString("hx1234"),
				},
				"_amount": 1566,
				"_data":   []byte{0x12, 0x34},
			},
			want: []interface{}{
				common.MustNewAddressFromString("hx1234"),
				1566,
				[]byte{0x12, 0x34},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			input := common.MustEncodeAny(tt.params)
			out, err := tt.method.EnsureParamsSequential(input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}
			expect := common.MustEncodeAny(tt.want)
			assert.EqualValues(t, expect, out)
		})
	}
}
