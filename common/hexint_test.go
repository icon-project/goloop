package common

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"reflect"
	"testing"

	"github.com/icon-project/goloop/common/codec"
)

func TestHexInt_UnmarshalJSON(t *testing.T) {
	type args struct {
		json string
	}
	longValue := new(big.Int)
	longValue.SetString("0x63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e", 0)

	tests := []struct {
		name     string
		args     args
		expected *big.Int
		error    bool
	}{
		{
			name:     "ShortNumber1",
			args:     args{"\"0x123\""},
			expected: big.NewInt(0x123),
			error:    false,
		},
		{
			name:     "ShortNumber2",
			args:     args{"291"},
			expected: big.NewInt(0x123),
			error:    false,
		},
		{
			name:     "ShortNumber3",
			args:     args{"\"-10\""},
			expected: big.NewInt(-0x10),
			error:    false,
		},
		{
			name:     "ShortNumber4",
			args:     args{"\"-0x80\""},
			expected: big.NewInt(-0x80),
			error:    false,
		},
		{
			name:     "ShortNumber5",
			args:     args{"\"0x80\""},
			expected: big.NewInt(0x80),
			error:    false,
		},
		{
			name:     "ShortFloat",
			args:     args{"291.5"},
			expected: nil,
			error:    true,
		},
		{
			name:     "LongNumber1",
			args:     args{"\"0x63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e\""},
			expected: longValue,
			error:    false,
		},
		{
			name:     "LongNumber1Err",
			args:     args{"\"x63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e\""},
			expected: nil,
			error:    true,
		},
		{
			name:     "JSON1Error",
			args:     args{"\"x63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e63546b8e"},
			expected: nil,
			error:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v1 HexInt
			if _, err := codec.JSON.UnmarshalFromBytes([]byte(tt.args.json), &v1); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't str=[%s]", tt.args.json)
					return
				}
			}

			if v1.Cmp(tt.expected) != 0 {
				t.Errorf("Invalid parsed value %s expected %s", v1.String(), tt.expected.String())
			}
		})
	}
}

func TestHexInt_EncodingDecoding(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Case1",
			args{"0x439394"},
			"439394",
		},
		{
			"Case2",
			args{"0x2"},
			"02",
		},
		{
			"Case3",
			args{"-0x1"},
			"ff",
		},
		{
			"Case4",
			args{"-0x80"},
			"80",
		},
		{
			"Case5",
			args{"0x80"},
			"0080",
		},
		{
			"Case6",
			args{"0x0"},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := hex.DecodeString(tt.want)
			if err != nil {
				return
			}

			var v1 HexInt
			v1.SetString(tt.args.s, 0)

			var delta HexInt
			delta.SetString("0x11223344556677889900", 0)
			v1.Int.Add(&v1.Int, &delta.Int)
			v1.Int.Sub(&v1.Int, &delta.Int)

			var b []byte
			b, err = codec.MarshalToBytes(&v1)
			if err != nil {
				t.Error(err)
				return
			}
			var b2 []byte
			b2, err = codec.MarshalToBytes(want)
			if !bytes.Equal(b2, b) {
				t.Errorf("Encoded = [%x] wanted = [%x]", b, want)
			}

			var v2 HexInt
			if _, err := codec.UnmarshalFromBytes(b, &v2); err != nil {
				t.Error(err)
				return
			}
			if v2.String() != tt.args.s {
				t.Errorf("Decoded = %s wanted = %s", v2.String(), tt.args.s)
			}
		})

	}
}

func TestHexInt16(t *testing.T) {
	type args struct {
		json  string
		value int16
	}
	tests := []struct {
		name  string
		args  args
		error bool
	}{
		{name: "JSON1", args: args{
			json:  "\"0x22\"",
			value: 0x22,
		}},
		{name: "JSON2", args: args{
			json:  "34",
			value: 0x22,
		}},
		{name: "JSON3", args: args{
			json:  "\"0x7fff\"",
			value: 0x7fff,
		}},
		{name: "JSON4Error", args: args{
			json:  "\"0x8080\"",
			value: 0,
		}, error: true},
		{name: "JSON5Error", args: args{
			json:  "\"0x80",
			value: 0,
		}, error: true},
		{name: "JSON6Error", args: args{
			json:  "\"0x80gt\"",
			value: 0,
		}, error: true},
		{name: "JSON7Error", args: args{
			json:  "\"cx2030\"",
			value: 0,
		}, error: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v1 HexInt16
			if _, err := codec.JSON.UnmarshalFromBytes([]byte(tt.args.json), &v1); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't str=[%s]", tt.args.json)
					return
				}
			}
			if v1.Value != tt.args.value {
				t.Errorf("Parsed value (%d) is different from (%d)", v1.Value, tt.args.value)
				return
			}

			b, err := codec.MarshalToBytes(&v1)
			if err != nil {
				t.Errorf("Encode fail with err=%+v", err)
				return
			}

			var v2 HexInt16
			if _, err := codec.UnmarshalFromBytes(b, &v2); err != nil {
				t.Errorf("Decode fail with err=%+v", err)
				return
			}

			if v2.Value != tt.args.value {
				t.Errorf("Decoded value (%d) is different from (%d)", v2.Value, tt.args.value)
			}
		})
	}
}

func TestHexInt32(t *testing.T) {
	type args struct {
		json  string
		value int32
	}
	tests := []struct {
		name  string
		args  args
		error bool
	}{
		{name: "JSON1", args: args{
			json:  "\"0x22\"",
			value: 0x22,
		}},
		{name: "JSON2", args: args{
			json:  "34",
			value: 0x22,
		}},
		{name: "JSON3", args: args{
			json:  "\"0x7fffffff\"",
			value: 0x7fffffff,
		}},
		{name: "JSON5Error", args: args{
			json:  "\"0x80",
			value: 0,
		}, error: true},
		{name: "JSON6Error", args: args{
			json:  "\"0x80gt\"",
			value: 0,
		}, error: true},
		{name: "JSON7Error", args: args{
			json:  "\"cx2030\"",
			value: 0,
		}, error: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v1 HexInt32

			if _, err := codec.JSON.UnmarshalFromBytes([]byte(tt.args.json), &v1); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't str=[%s]", tt.args.json)
					return
				}
			}

			if v1.Value != tt.args.value {
				t.Errorf("Parsed value (%d) is different from (%d)", v1.Value, tt.args.value)
				return
			}

			b, err := codec.MarshalToBytes(&v1)
			if err != nil {
				t.Errorf("Encode fail with err=%v", err)
				return
			}

			var v2 HexInt32
			if _, err := codec.UnmarshalFromBytes(b, &v2); err != nil {
				t.Errorf("Decode fail with err=%v", err)
				return
			}

			if v2.Value != tt.args.value {
				t.Errorf("Decoded value (%d) is different from (%d)", v2.Value, tt.args.value)
			}
		})
	}
}

func TestHexInt64(t *testing.T) {
	type args struct {
		json  string
		value int64
	}
	tests := []struct {
		name  string
		args  args
		error bool
	}{
		{name: "JSON1", args: args{
			json:  "\"0x22\"",
			value: 0x22,
		}},
		{name: "JSON2", args: args{
			json:  "34",
			value: 0x22,
		}},
		{name: "JSON3", args: args{
			json:  "\"0x7fffffffffffffff\"",
			value: 0x7fffffffffffffff,
		}},
		{name: "JSON4Error", args: args{
			json:  "\"0xffffffffffffffff\"",
			value: 0,
		}, error: true},
		{name: "JSON5Error", args: args{
			json:  "\"0x80",
			value: 0,
		}, error: true},
		{name: "JSON6Error", args: args{
			json:  "\"0x80gt\"",
			value: 0,
		}, error: true},
		{name: "JSON7Error", args: args{
			json:  "\"cx2030\"",
			value: 0,
		}, error: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v1 HexInt64
			if _, err := codec.JSON.UnmarshalFromBytes([]byte(tt.args.json), &v1); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't str=[%s]", tt.args.json)
					return
				}
			}

			if v1.Value != tt.args.value {
				t.Errorf("Parsed value (%d) is different from (%d)", v1.Value, tt.args.value)
				return
			}

			b, err := codec.MarshalToBytes(&v1)
			if err != nil {
				t.Errorf("Encode fail with err=%v", err)
				return
			}

			var v2 HexInt64
			if _, err := codec.UnmarshalFromBytes(b, &v2); err != nil {
				t.Errorf("Decode fail with err=%v", err)
				return
			}

			if v2.Value != tt.args.value {
				t.Errorf("Decoded value (%d) is different from (%d)", v2.Value, tt.args.value)
			}
		})
	}
}

func TestHexUint16(t *testing.T) {
	type args struct {
		json  string
		value uint16
	}
	tests := []struct {
		name  string
		args  args
		error bool
	}{
		{name: "JSON1", args: args{
			json:  "\"0x22\"",
			value: 0x22,
		}},
		{name: "JSON2", args: args{
			json:  "34",
			value: 0x22,
		}},
		{name: "JSON3", args: args{
			json:  "\"0xffff\"",
			value: 0xffff,
		}},
		{name: "JSON4Error", args: args{
			json:  "\"0xffffff\"",
			value: 0,
		}, error: true},
		{name: "JSON5Error", args: args{
			json:  "\"0x80",
			value: 0,
		}, error: true},
		{name: "JSON6Error", args: args{
			json:  "\"0x80gt\"",
			value: 0,
		}, error: true},
		{name: "JSON7Error", args: args{
			json:  "\"cx2030\"",
			value: 0,
		}, error: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v1 HexUint16
			if _, err := codec.JSON.UnmarshalFromBytes([]byte(tt.args.json), &v1); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't str=[%s]", tt.args.json)
					return
				}
			}

			if v1.Value != tt.args.value {
				t.Errorf("Parsed value (%d) is different from (%d)", v1.Value, tt.args.value)
				return
			}

			b, err := codec.MarshalToBytes(&v1)
			if err != nil {
				t.Errorf("Encode fail with err=%+v", err)
				return
			}

			var v2 HexUint16
			if _, err := codec.UnmarshalFromBytes(b, &v2); err != nil {
				t.Errorf("Decode fail with err=%v", err)
				return
			}

			if v2.Value != tt.args.value {
				t.Errorf("Decoded value (%d) is different from (%d)", v2.Value, tt.args.value)
			}
		})
	}
}

func TestHexUint32(t *testing.T) {
	type args struct {
		json  string
		value uint32
	}
	tests := []struct {
		name  string
		args  args
		error bool
	}{
		{name: "JSON1", args: args{
			json:  "\"0x22\"",
			value: 0x22,
		}},
		{name: "JSON2", args: args{
			json:  "34",
			value: 0x22,
		}},
		{name: "JSON3", args: args{
			json:  "\"0xffffffff\"",
			value: 0xffffffff,
		}},
		{name: "JSON4Error", args: args{
			json:  "\"0xffffffffff\"",
			value: 0,
		}, error: true},
		{name: "JSON5Error", args: args{
			json:  "\"0x80",
			value: 0,
		}, error: true},
		{name: "JSON6Error", args: args{
			json:  "\"0x80gt\"",
			value: 0,
		}, error: true},
		{name: "JSON7Error", args: args{
			json:  "\"cx2030\"",
			value: 0,
		}, error: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v1 HexUint32
			if _, err := codec.JSON.UnmarshalFromBytes([]byte(tt.args.json), &v1); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't str=[%s]", tt.args.json)
					return
				}
			}

			if v1.Value != tt.args.value {
				t.Errorf("Parsed value (%d) is different from (%d)", v1.Value, tt.args.value)
				return
			}

			b, err := codec.MarshalToBytes(&v1)
			if err != nil {
				t.Errorf("Encode fail with err=%v", err)
				return
			}

			var v2 HexUint32
			if _, err := codec.UnmarshalFromBytes(b, &v2); err != nil {
				t.Errorf("Decode fail with err=%v", err)
				return
			}

			if v2.Value != tt.args.value {
				t.Errorf("Decoded value (%d) is different from (%d)", v2.Value, tt.args.value)
			}
		})
	}
}

func TestHexUint64(t *testing.T) {
	type args struct {
		json  string
		value uint64
	}
	tests := []struct {
		name  string
		args  args
		error bool
	}{
		{name: "JSON1", args: args{
			json:  "\"0x22\"",
			value: 0x22,
		}},
		{name: "JSON2", args: args{
			json:  "34",
			value: 0x22,
		}},
		{name: "JSON3Max", args: args{
			json:  "\"0xffffffffffffffff\"",
			value: 0xffffffffffffffff,
		}},
		{name: "JSON5Error", args: args{
			json:  "\"0x80",
			value: 0,
		}, error: true},
		{name: "JSON6Error", args: args{
			json:  "\"0x80gt\"",
			value: 0,
		}, error: true},
		{name: "JSON7Error", args: args{
			json:  "\"cx2030\"",
			value: 0,
		}, error: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v1 HexUint64
			if _, err := codec.JSON.UnmarshalFromBytes([]byte(tt.args.json), &v1); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't str=[%s]", tt.args.json)
					return
				}
			}

			if v1.Value != tt.args.value {
				t.Errorf("Parsed value (%d) is different from (%d)", v1.Value, tt.args.value)
				return
			}

			b, err := codec.MarshalToBytes(&v1)
			if err != nil {
				t.Errorf("Encode fail with err=%v", err)
				return
			}

			var v2 HexUint64
			if _, err := codec.UnmarshalFromBytes(b, &v2); err != nil {
				t.Errorf("Decode fail with err=%v", err)
				return
			}

			if v2.Value != tt.args.value {
				t.Errorf("Decoded value (%d) is different from (%d)", v2.Value, tt.args.value)
			}
		})
	}
}

func TestBigIntToBytes(t *testing.T) {
	type args struct {
		i *big.Int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"T1", args{big.NewInt(-0x1)}, []byte{0xff}},
		{"T2", args{big.NewInt(-0x7f)}, []byte{0x81}},
		{"T3", args{big.NewInt(0x80)}, []byte{0x00, 0x80}},
		{"T4", args{big.NewInt(-0x80)}, []byte{0x80}},
		{"T5", args{big.NewInt(0)}, []byte{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BigIntToBytes(tt.args.i); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BigIntToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt64ToBytes(t *testing.T) {
	type args struct {
		v int64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"T1", args{-1}, []byte{0xff}},
		{"T2", args{-0x7f}, []byte{0x81}},
		{"T3", args{0x80}, []byte{0x00, 0x80}},
		{"T4", args{-0x80}, []byte{0x80}},
		{"T5", args{0}, []byte{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int64ToBytes(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Int64ToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBytesToInt64(t *testing.T) {
	type args struct {
		bs []byte
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{"T1", args{[]byte{}}, 0},
		{"T2", args{[]byte{0x80}}, -0x80},
		{"T3", args{[]byte{0x00, 0x80}}, 0x80},
		{"T4", args{[]byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}, -0x8000000000000000},
		{"T5", args{[]byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}}, -0x7fffffffffffffff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BytesToInt64(tt.args.bs); got != tt.want {
				t.Errorf("BytesToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseUint(t *testing.T) {
	type args struct {
		s    string
		bits int
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{"T1", args{"0x0", 16}, 0, false},
		{"T2", args{"0xffff", 16}, 0xffff, false},
		{"T3", args{"0xffffffffffffffff", 64}, 0xffffffffffffffff, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUint(tt.args.s, tt.args.bits)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseUint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	type args struct {
		s    string
		bits int
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{"T1", args{"0x0", 16}, 0, false},
		{"T2", args{"0x7fff", 16}, 0x7fff, false},
		{"T3", args{"-0x8000", 16}, -0x8000, false},
		{"T4", args{"0xffff", 16}, 0, true},
		{"T5", args{"0x0ffff", 16}, 0, true},
		{"T6", args{"-0x8000000000000000", 64}, -0x8000000000000000, false},
		{"T7", args{"-0x10000000000000000", 64}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInt(tt.args.s, tt.args.bits)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatInt(t *testing.T) {
	type args struct {
		v int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"T0", args{0x00}, "0x0"},
		{"T1", args{-0x1}, "-0x1"},
		{"T2", args{-0x80}, "-0x80"},
		{"T3", args{0x80}, "0x80"},
		{"T4", args{-0xff}, "-0xff"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatInt(tt.args.v); got != tt.want {
				t.Errorf("FormatInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
