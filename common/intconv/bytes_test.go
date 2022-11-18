package intconv

import (
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		{"T5", args{big.NewInt(0)}, []byte{0x00}},
		{"T6", args{big.NewInt(0x0214)}, []byte{0x02, 0x14}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BigIntToBytes(tt.args.i); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BigIntToBytes() = %v, want %v", got, tt.want)
			} else {
				i2 := BigIntSetBytes(new(big.Int), got)
				assert.Equal(t, 0, tt.args.i.Cmp(i2))
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
		{"T5", args{0}, []byte{0x00}},
		{"T6", args{-0x8000000000000000}, []byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"T7", args{0x7fffffffffffffff}, []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int64ToBytes(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Int64ToBytes() = %v, want %v", got, tt.want)
			} else {
				value := BytesToInt64(got)
				assert.Equal(t, tt.args.v, value)
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
		fail bool
	}{
		{"T1", args{[]byte{}}, 0, false},
		{"T2", args{[]byte{0x80}}, -0x80, false},
		{"T3", args{[]byte{0x00, 0x80}}, 0x80, false},
		{"T4", args{[]byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}, -0x8000000000000000, false},
		{"T5", args{[]byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}}, -0x7fffffffffffffff, false},
		{"T6", args{[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}, 0, true},
		{"T7", args{[]byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); tt.fail != (r != nil) {
					if tt.fail {
						t.Error("Expecting failure")
					} else {
						t.Error("Expecting Success")
					}
				}
			}()
			if got := BytesToInt64(tt.args.bs); got != tt.want {
				t.Errorf("BytesToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUint64ToBytes(t *testing.T) {
	type args struct {
		v uint64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"T1", args{0x00}, []byte{0x00}},
		{"T2", args{0x80}, []byte{0x00, 0x80}},
		{"T3", args{0x80123456789abcde}, []byte{0x00, 0x80, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde}},
		{"T4", args{0x7fffffffffffffff}, []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{"T5", args{0xffffffffffffffff}, []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Uint64ToBytes(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Uint64ToBytes() = %v, want %v", got, tt.want)
			} else {
				value := BytesToUint64(got)
				assert.Equal(t, tt.args.v, value)
			}
		})
	}
}

func TestBytesToUint64(t *testing.T) {
	type args struct {
		bs []byte
	}
	tests := []struct {
		name string
		args args
		want uint64
		fail bool
	}{
		{"T0", args{nil}, 0, false},
		{"T1", args{[]byte{}}, 0, false},
		{"T2", args{[]byte{0x80}}, 0, true},
		{"T3", args{[]byte{0x00, 0x80}}, 0x80, false},
		{"T4", args{[]byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}, 0x0, true},
		{"T5", args{[]byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, 0x7fffffffffffffff, false},
		{"T6", args{[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, 0x0, true},
		{"T7", args{[]byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, 0xffffffffffffffff, false},
		{"T8", args{[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}, 0x0, true},
		{"T9", args{[]byte{0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, 0x0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if e := recover(); tt.fail != (e != nil) {
					if tt.fail {
						t.Error("Expecting failure")
					} else {
						t.Error("Expecting success")
					}
				}
			}()
			if got := BytesToUint64(tt.args.bs); got != tt.want {
				t.Errorf("BytesToUint64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeBytesToSize64(t *testing.T) {
	type args struct {
		bs []byte
	}
	tests := []struct {
		name  string
		args  args
		want  uint64
		want1 bool
	}{
		{"Nil", args{nil}, 0, true},
		{"Empty", args{[]byte{}}, 0, true},
		{"Max", args{[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, 0xffffffffffffffff, true},
		{"7Bytes", args{[]byte{0x87, 0x65, 0x43, 0x21, 0xfe, 0xdc, 0xba}}, 0x87654321fedcba, true},
		{"6Bytes", args{[]byte{0x87, 0x65, 0x43, 0x21, 0xfe, 0xdc}}, 0x87654321fedc, true},
		{"5Bytes", args{[]byte{0x87, 0x65, 0x43, 0x21, 0xfe}}, 0x87654321fe, true},
		{"4Bytes", args{[]byte{0x87, 0x65, 0x43, 0x21}}, 0x87654321, true},
		{"3Bytes", args{[]byte{0x87, 0x65, 0x43}}, 0x876543, true},
		{"2Bytes", args{[]byte{0x87, 0x65}}, 0x8765, true},
		{"1Bytes", args{[]byte{0x87}}, 0x87, true},
		{"OverFlow", args{[]byte{0x87, 0x65, 0x43, 0x21, 0xfe, 0xdc, 0xba, 0x98, 0x76}}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value1, ok1 := SafeBytesToSize64(tt.args.bs)
			assert.Equalf(t, tt.want, value1, "SafeBytesToSize64(%#x)", tt.args.bs)
			assert.Equalf(t, tt.want1, ok1, "SafeBytesToSize64(%#x)", tt.args.bs)
			value2, ok2 := SafeBytesToSize(tt.args.bs)
			if value1 > math.MaxInt {
				assert.False(t, ok2)
				assert.Zero(t, value2)
			} else {
				assert.Equal(t, value1, uint64(value2))
				assert.Equal(t, ok1, ok2)
			}
		})
	}
}
