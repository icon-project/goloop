package intconv

import (
	"math/big"
	"testing"
)

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
		{"T4", args{"0x01ffffffffffffffff", 64}, 0, true},
		{"T5", args{"-0x1", 64}, 0, true},
		{"T6", args{"0x00ffffffffffffffff", 64}, 0xffffffffffffffff, false},
		{"T7", args{"1556309404241523", 64}, 1556309404241523, false},
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
		{"T8", args{"0x7fffffffffffffff", 64}, 0x7fffffffffffffff, false},
		{"T9", args{"0x07fff", 16}, 0x7fff, false},
		{"T10", args{"0x07fffffff", 32}, 0x7fffffff, false},
		{"T11", args{"0x07fffffffffffffff", 64}, 0x7fffffffffffffff, false},
		{"T12", args{"1556309404241523", 64}, 1556309404241523, false},
		{"T13", args{"0x", 8}, 0, true},
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

func TestParseBigInt(t *testing.T) {
	type args struct {
		s string
	}
	n1 := new(big.Int).SetBytes([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	n1.Neg(n1)

	tests := []struct {
		name    string
		args    args
		want    *big.Int
		wantErr bool
	}{
		{"T1", args{"0x0"}, big.NewInt(0), false},
		{"T2", args{"0x7fff"}, big.NewInt(0x7fff), false},
		{"T3", args{"-0x8000000000000000"}, big.NewInt(-0x8000000000000000), false},
		{"T4", args{"-0x10000000000000000"}, n1, false},
		{"T5", args{"-18446744073709551616"}, n1, false},
		{"T6", args{"-1844674407370955161a"}, nil, true},
		{"T7", args{"887234"}, big.NewInt(887234), false},
		{"T8E", args{"0x-b"}, nil, true},
		{"T9E", args{"0x-0"}, nil, true},
		{"T10", args{"0x_1_1"}, big.NewInt(0x11), false},
		{"T11", args{"10_000"}, big.NewInt(10_000), false},
		{"T12E", args{"10__000"}, nil, true},
		{"T13E", args{"10_000_"}, nil, true},
		{"T14E", args{"0b00"}, nil, true},
		{"T15E", args{"0o12"}, nil, true},
		{"T14E", args{"0B00"}, nil, true},
		{"T15E", args{"0O12"}, nil, true},
		{"T16E", args{"0X12"}, nil, true},
		{"T17", args{"0700"}, big.NewInt(700), false},
		{"T18", args{"0_700"}, big.NewInt(700), false},
		{"T19", args{"-0_700"}, big.NewInt(-700), false},
		{"T20E", args{"-_700"}, nil, true},
		{"T21", args{"-0"}, big.NewInt(0), false},
		{"T22E", args{"-0__100"}, nil, true},
		{"T23E", args{"-0_100_"}, nil, true},
		{"T24E", args{"19928560000000000000x0"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got big.Int
			err := ParseBigInt(&got, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBigInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if tt.want.Cmp(&got) != 0 {
					t.Errorf("ParseBigInt() = %v, want %v", &got, tt.want)
				}
			} else {
				t.Logf("Expected error for %q err=%v", tt.args.s, err)
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
		{"T5", args{-0x8000000000000000}, "-0x8000000000000000"},
		{"T6", args{0x7f7f7f7f7f7f7f7f}, "0x7f7f7f7f7f7f7f7f"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatInt(tt.args.v); got != tt.want {
				t.Errorf("FormatInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
