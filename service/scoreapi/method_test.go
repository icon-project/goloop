package scoreapi

import (
	"reflect"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
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
				common.NewAddressFromString("hxff9221db215ce1a511cbe0a12ff9eb70be4e5764"),
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
				common.NewAddressFromString("hxff9221db215ce1a511cbe0a12ff9eb70be4e5764"),
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
			got, err := a.ConvertParamsToTypedObj(tt.args.bs)
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
