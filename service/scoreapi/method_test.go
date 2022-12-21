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
			"SimpleReader",
			&Method{
				Type:    Function,
				Name:    "balanceOf",
				Flags:   FlagExternal | FlagReadOnly | FlagIsolated,
				Indexed: 1,
				Inputs: []Parameter{
					{
						Name: "_addr",
						Type: Address,
					},
				},
				Outputs: []DataType{Integer},
			},
			"balanceOf(Address)",
			false,
			map[string]interface{}{
				"type": "function",
				"name": "balanceOf",
				"inputs": []interface{}{
					map[string]interface{}{
						"name": "_addr",
						"type": "Address",
					},
				},
				"outputs": []interface{}{
					map[string]interface{}{
						"type": "int",
					},
				},
				"readonly": "0x1",
				"isolated": "0x1",
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
		{
			name:     "list",
			dtString: "list",
			dtValue:  List,
		},
		{
			name:     "dict",
			dtString: "dict",
			dtValue:  Dict,
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

func TestDataType_ConvertBytesToTypedObj(t *testing.T) {
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

func TestMethod_EnsureParamsSequential(t *testing.T) {
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
func TestDataType_ValidateSome(t *testing.T) {
	type eventBytes struct {
		bytes   []byte
		wantErr bool
	}
	type outputObj struct {
		obj     *codec.TypedObj
		wantErr bool
	}
	type inputObj struct {
		obj      *codec.TypedObj
		nullable bool
		wantErr  bool
	}
	var cases = []struct {
		name    string
		dt      DataType
		fields  []Field
		events  []eventBytes
		outputs []outputObj
		inputs  []inputObj
	}{
		{
			name: "Integer",
			dt:   Integer,
			events: []eventBytes{
				{[]byte{0x80}, false},
				{nil, false},
				{[]byte{}, true},
			},
			outputs: []outputObj{
				{common.MustEncodeAny(1234), false},
				{common.MustEncodeAny("string"), true},
				{common.MustEncodeAny(nil), true},
			},
			inputs: []inputObj{
				{common.MustEncodeAny(1234), false, false},
				{common.MustEncodeAny("string"), false, true},
				{common.MustEncodeAny(nil), false, true},
				{nil, true, false},
			},
		},
		{
			name: "String",
			dt:   String,
			events: []eventBytes{
				{[]byte("hello"), false},
				{[]byte{0x00}, false},
				{[]byte{0x89}, true},
				{[]byte{}, false},
			},
			outputs: []outputObj{
				{common.MustEncodeAny("string"), false},
				{common.MustEncodeAny(1234), true},
				{nil, true},
			},
			inputs: []inputObj{
				{common.MustEncodeAny("string"), false, false},
				{common.MustEncodeAny(1234), false, true},
			},
		},
		{
			name: "Bool",
			dt:   Bool,
			events: []eventBytes{
				{[]byte{0x01}, false},
				{[]byte{}, true},
				{[]byte{0x80}, true},
				{[]byte{0x00, 0x01}, true},
			},
			outputs: []outputObj{
				{common.MustEncodeAny(true), false},
				{common.MustEncodeAny(1234), true},
				{nil, true},
			},
			inputs: []inputObj{
				{common.MustEncodeAny(true), false, false},
				{common.MustEncodeAny(1234), false, true},
			},
		},
		{
			name: "Address",
			dt:   Address,
			events: []eventBytes{
				{[]byte("\x00\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12"), false},
				{[]byte("hello"), true},
				{[]byte{0x00}, true},
				{[]byte{}, true},
			},
			outputs: []outputObj{
				{common.MustEncodeAny(common.MustNewAddressFromString("cx00")), false},
				{common.MustEncodeAny(1234), true},
				{nil, false},
			},
			inputs: []inputObj{
				{common.MustEncodeAny(common.MustNewAddressFromString("cx00")), false, false},
				{common.MustEncodeAny(1234), false, true},
				{nil, true, false},
			},
		},
		{
			name: "Bytes",
			dt:   Bytes,
			events: []eventBytes{
				{[]byte("\x00\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12\x12"), false},
				{[]byte("hello"), false},
				{[]byte{0x00}, false},
				{[]byte{}, false},
			},
			outputs: []outputObj{
				{common.MustEncodeAny([]byte{0x12, 0x34}), false},
				{common.MustEncodeAny(1234), true},
				{nil, false},
			},
			inputs: []inputObj{
				{common.MustEncodeAny([]byte{0x12, 0x34}), false, false},
				{common.MustEncodeAny(1234), false, true},
				{nil, false, true},
			},
		},
		{
			name: "Struct",
			dt:   Struct,
			fields: []Field{
				{
					Name: "name",
					Type: String,
				},
				{
					Name: "address",
					Type: Address,
				},
			},
			events: []eventBytes{
				{[]byte{0x12, 0x34}, true},
				{[]byte{}, true},
				{nil, true},
			},
			outputs: []outputObj{
				{common.MustEncodeAny([]byte{0x12, 0x34}), true},
				{common.MustEncodeAny(1234), true},
				{common.MustEncodeAny(map[string]interface{}{"key1": "value1"}), true},
				{nil, true},
			},
			inputs: []inputObj{
				{common.MustEncodeAny(map[string]interface{}{
					"name":    "MyName",
					"address": common.MustNewAddressFromString("hx1234"),
				}), false, false},
				{common.MustEncodeAny(map[string]interface{}{
					"name": "MyName",
				}), false, true},
				{common.MustEncodeAny(map[string]interface{}{
					"name":    "MyName",
					"address": common.MustNewAddressFromString("hx1234"),
					"value":   1,
				}), false, true},
				{common.MustEncodeAny([]byte{0x12, 0x34}), false, true},
				{common.MustEncodeAny(1234), false, true},
				{common.MustEncodeAny(map[string]interface{}{"key1": "value1"}), false, true},
				{nil, false, true},
			},
		},
		{
			name: "ListOfBytes",
			dt:   ListTypeOf(1, Bytes),
			events: []eventBytes{
				{[]byte{0x12, 0x34}, true},
				{[]byte{}, true},
				{nil, true},
			},
			outputs: []outputObj{
				{common.MustEncodeAny([]byte{0x12, 0x34}), true},
				{common.MustEncodeAny(1234), true},
				{nil, true},
			},
			inputs: []inputObj{
				{common.MustEncodeAny([]interface{}{[]byte{0x12, 0x34}, []byte{0x56, 0x78}}), false, false},
				{common.MustEncodeAny([]byte{0x12, 0x34}), false, true},
				{common.MustEncodeAny(1234), false, true},
				{nil, false, true},
			},
		},
		{
			name: "Dict",
			dt:   Dict,
			events: []eventBytes{
				{[]byte{0x12, 0x34}, true},
				{[]byte{}, true},
				{nil, true},
			},
			outputs: []outputObj{
				{common.MustEncodeAny(map[string]interface{}{"key1": "value1"}), false},
				{common.MustEncodeAny([]byte{0x12, 0x34}), true},
				{common.MustEncodeAny(1234), true},
				{nil, false},
			},
			inputs: []inputObj{
				{common.MustEncodeAny([]byte{0x12, 0x34}), false, true},
				{nil, true, true},
			},
		},
		{
			name: "List",
			dt:   List,
			events: []eventBytes{
				{[]byte{0x12, 0x34}, true},
				{[]byte{}, true},
				{nil, true},
			},
			outputs: []outputObj{
				{common.MustEncodeAny([]interface{}{"key1", 123}), false},
				{common.MustEncodeAny([]byte{0x12, 0x34}), true},
				{common.MustEncodeAny(1234), true},
				{nil, false},
			},
			inputs: []inputObj{
				{common.MustEncodeAny([]interface{}{0x12, 0x34}), false, true},
				{nil, true, true},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for _, event := range tt.events {
				err := tt.dt.ValidateEvent(event.bytes)
				if event.wantErr {
					assert.Errorf(t, err, "Unexpected success for value=%#x", event.bytes)
				} else {
					assert.NoErrorf(t, err, "Unexpected error for value=%#x err=%+v", event.bytes, err)
					_, err = tt.dt.ConvertBytesToJSO(event.bytes)
					assert.NoError(t, err)
				}
			}
			for i, output := range tt.outputs {
				err := tt.dt.ValidateOutput(output.obj)
				if output.wantErr {
					assert.Errorf(t, err, "Unexpected success for outputs[%d]", i)
				} else {
					assert.NoErrorf(t, err, "Unexpected error for outputs[%d] err=%+v", i, err)
				}
			}

			for i, input := range tt.inputs {
				err := tt.dt.ValidateInput(input.obj, tt.fields, input.nullable)
				if input.wantErr {
					assert.Errorf(t, err, "Unexpected success for inputs[%d]", i)
				} else {
					assert.NoErrorf(t, err, "Unexpected error for inputs[%d] err=%+v", i, err)
				}
			}
		})
	}
}

func TestDataType_ConvertJSONToTypedObj(t *testing.T) {
	type jsonToObj struct {
		json    string
		wantErr bool
		want    *codec.TypedObj
	}
	var cases = []struct {
		name     string
		dt       DataType
		fields   []Field
		nullable bool
		jsons    []jsonToObj
	}{
		{
			name: "Integer",
			dt:   Integer,
			jsons: []jsonToObj{
				{`"0x123"`, false, common.MustEncodeAny(0x123)},
				{`"abc"`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name: "String",
			dt:   String,
			jsons: []jsonToObj{
				{`"0x123"`, false, common.MustEncodeAny("0x123")},
				{`12`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name: "Bool",
			dt:   Bool,
			jsons: []jsonToObj{
				{`"0x1"`, false, common.MustEncodeAny(true)},
				{`"0x3"`, true, nil},
				{`true`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name: "Address",
			dt:   Address,
			jsons: []jsonToObj{
				{`"cx0000000000000000000000000000000000000000"`, false, common.MustEncodeAny(common.MustNewAddressFromString("cx00"))},
				{`"cxb"`, true, nil},
				{`12`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name: "Bytes",
			dt:   Bytes,
			jsons: []jsonToObj{
				{`"0x12bc"`, false, common.MustEncodeAny([]byte{0x12, 0xbc})},
				{`"0x2"`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name: "Struct",
			dt:   Struct,
			fields: []Field{
				{
					Name: "name",
					Type: String,
				},
				{
					Name: "address",
					Type: Address,
				},
			},
			jsons: []jsonToObj{
				{`{ "name": "0x1234", "address": "hx12ff000000000000000000000000000000000000" }`, false,
					common.MustEncodeAny(map[string]interface{}{
						"name":    "0x1234",
						"address": common.MustNewAddressFromString("hx12ff000000000000000000000000000000000000"),
					})},
				{`{ "name": "0x1234", "address": 1234 }`, true, nil},
				{`{ "name": "0x1234", "addr2": 1234 }`, true, nil},
				{`"0x2"`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name: "ListOfBytes",
			dt:   ListTypeOf(1, Bytes),
			jsons: []jsonToObj{
				{`[ "0x12", "0x34" ]`, false, common.MustEncodeAny([]interface{}{[]byte{0x12}, []byte{0x34}})},
				{`[ "0x12", 0x34 ]`, true, nil},
				{`"0x2"`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name: "Dict",
			dt:   Dict,
			jsons: []jsonToObj{
				{`{ "name": "0x1234", "address": "hx12ff000000000000000000000000000000000000" }`, true, nil},
				{`"0x2"`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name: "List",
			dt:   List,
			jsons: []jsonToObj{
				{`[ "0x12", "0x34" ]`, true, nil},
				{`"0x2"`, true, nil},
				{`null`, true, nil},
			},
		},
		{
			name:     "StringNullable",
			dt:       String,
			nullable: true,
			jsons: []jsonToObj{
				{`null`, false, common.MustEncodeAny(nil)},
				{`123`, true, nil},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for _, jso := range tt.jsons {
				obj, err := tt.dt.ConvertJSONToTypedObj([]byte(jso.json), tt.fields, tt.nullable)
				if jso.wantErr {
					assert.Errorf(t, err, "Unexpected success for js=%s", jso.json)
					assert.Nil(t, obj)
				} else {
					assert.NoErrorf(t, err, "Unexpected error for js=%s err=%+v", jso.json, err)
					assert.EqualValues(t, jso.want, obj)
				}
			}
		})
	}
}
