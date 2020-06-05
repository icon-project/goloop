package scoreapi

import (
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
