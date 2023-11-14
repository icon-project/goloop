package icutils

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
)

func TestPow10(t *testing.T) {
	expected := int64(1)
	for i := 0; i < 19; i++ {
		assert.Zero(t, big.NewInt(expected).Cmp(Pow10(i)))
		expected *= 10
	}

	assert.Nil(t, Pow10(-1))
}

func TestToDecimal(t *testing.T) {
	nums := []int{1, -2, 0}

	for _, x := range nums {
		expected := int64(x)
		for i := 0; i < 10; i++ {
			d := ToDecimal(x, i)
			assert.Equal(t, expected, d.Int64())
			expected *= 10
		}
	}

	assert.Nil(t, ToDecimal(1, -10))
}

func TestToLoop(t *testing.T) {
	var expected int64
	nums := []int{0, 1, -1}

	for _, x := range nums {
		expected = int64(x) * 1_000_000_000_000_000_000
		assert.Zero(t, big.NewInt(expected).Cmp(ToDecimal(x, 18)))
	}
}

func TestValidateRange(t *testing.T) {

	type args struct {
		old    *big.Int
		new    *big.Int
		minPct int
		maxPct int
	}

	tests := []struct {
		name string
		in   args
		err  bool
	}{
		{
			"OK",
			args{
				new(big.Int).SetInt64(100),
				new(big.Int).SetInt64(110),
				20,
				20,
			},
			false,
		},
		{
			"Same value",
			args{
				new(big.Int).SetInt64(100),
				new(big.Int).SetInt64(100),
				20,
				20,
			},
			false,
		},
		{
			"Old is Zero",
			args{
				new(big.Int).SetInt64(0),
				new(big.Int).SetInt64(10),
				20,
				20,
			},
			true,
		},
		{
			"New is Zero",
			args{
				new(big.Int).SetInt64(10),
				new(big.Int).SetInt64(0),
				20,
				20,
			},
			true,
		},
		{
			"too small",
			args{
				new(big.Int).SetInt64(100),
				new(big.Int).SetInt64(79),
				20,
				20,
			},
			true,
		},
		{
			"too big",
			args{
				new(big.Int).SetInt64(100),
				new(big.Int).SetInt64(121),
				20,
				20,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			err := ValidateRange(in.old, in.new, in.minPct, in.maxPct)
			if tt.err {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEndpoint(t *testing.T) {
	shouldMatch := []string{
		"foo.com:1", "192.10.6.2:8000", "localhost:1234", "127.0.0.12:7100",
	}

	for _, x := range shouldMatch {
		t.Run(x, func(t *testing.T) {
			err := ValidateEndpoint(x)
			assert.NoError(t, err)
		})
	}

	shouldFail := []string{
		"http://", "http://.", "http://..", "http://../", "http://?", "http://??", "http://??/",
		"http://#", "http://##", "http://##/", "http://foo.bar?q=Spaces should be encoded",
		"//", "//a", "///a", "///", "http:///a", "foo.com", "rdar://1234", "h://test",
		"http:// shouldfail.com", "http://foo.bar/foo(bar)baz quux", "ftps://foo.bar/",
		"http://-error-.invalid/", "http://-a.b.co", "http://a.b-.co", "http://0.0.0.0:8080",
		"http://3628126748", "http://.www.foo.bar/", "http://www.foo.bar./",
		"http://.www.foo.bar./", "http://:8080", "http://.:8080", "http://..:8080",
		"http://../:8080", "http://?:8080", "http://??:8080", "http://??/:8080", "http://#:8080",
		"http://##:8080", "http://##/:8080", "http://foo.bar?q=Spaces should be encoded:8080",
		"//:8080", "//a:8080", "///a:8080", "///:8080", "http:///a:8080", "rdar://1234:8080",
		"h://test:8080", "http:// shouldfail.com:8080", "http://foo.bar/foo(bar)baz quux:8080",
		"ftps://foo.bar/:8080", "http://-error-.invalid/:8080", "http://-a.b.co:8080",
		"http://a.b-.co:8080", "http://3628126748:8080", "http://.www.foo.bar/:8080",
		"http://www.foo.bar./:8080", "http://.www.foo.bar./:8080",
		".www.goo.bar:8080", "9.127.0.0.1:9090",
	}

	for _, x := range shouldFail {
		t.Run(x, func(t *testing.T) {
			err := ValidateEndpoint(x)
			assert.Error(t, err)
		})
	}
}

func TestValidateURL(t *testing.T) {
	shouldMatch := []string{
		"http://foo.com/blah_blah", "http://foo.com/blah_blah/", "http://foo.com/blah_blah_(wikipedia)",
		"http://foo.com/blah_blah_(wikipedia)_(again)", "http://www.example.com/wpstyle/?p=364",
		"https://www.example.com/foo/?bar=baz&inga=42&quux", "http://odf.ws/123",
		"http://foo.com/blah_(wikipedia)#cite-1", "http://foo.com/blah_(wikipedia)_blah#cite-1",
		"http://foo.com/unicode_(✪)_in_parens", "http://foo.com/(something)?after=parens",
		"http://code.google.com/events/#&product=browser", "http://foo.bar/?q=Test%20URL-encoded%20stuff",
		"http://1337.net", "http://223.255.255.254", "http://foo.bar:8080", "https://foo.bar:8000",
		"https://localhost:1234", "http://localhost:1234", "http://localhost", "https://localhost",
	}

	for _, x := range shouldMatch {
		t.Run(x, func(t *testing.T) {
			err := ValidateURL(x)
			assert.NoError(t, err)
		})
	}

	shouldFail := []string{
		"http://", "http://.", "http://..", "http://../", "http://?", "http://??", "http://??/",
		"http://#", "http://##", "http://##/", "http://foo.bar?q=Spaces should be encoded",
		"//", "//a", "///a", "///", "http:///a", "foo.com", "rdar://1234", "h://test",
		"http:// shouldfail.com", "http://foo.bar/foo(bar)baz quux", "ftps://foo.bar/",
		"http://-error-.invalid/", "http://-a.b.co", "http://a.b-.co", "http://3628126748",
		"http://.www.foo.bar/", "http://www.foo.bar./", "http://.www.foo.bar./",
		"http://022.107.254.1",
	}

	for _, x := range shouldFail {
		t.Run(x, func(t *testing.T) {
			err := ValidateURL(x)
			assert.Error(t, err)
		})
	}

}

func TestValidateEmail(t *testing.T) {
	shouldFailBeforeRev9 := []string{
		"invalid email", "invalid.com", "invalid@", "invalid@a",
		"invalid@a.", "invalid@.com", "invalid.@asdf.com-",
		"email@domain..com", "john..doe@example.com", ".invalid@email.com",
	}

	for _, x := range shouldFailBeforeRev9 {
		t.Run(x, func(t *testing.T) {
			err := ValidateEmail(x, 5)
			assert.Error(t, err)
		})
	}

	e := 253
	s1 := ""
	for i := 0; i < e; i++ {
		s1 += "a"
	}

	e = 64
	s2 := ""
	for i := 0; i < e; i++ {
		s2 += "가"
	}

	shouldFailAfterRev9 := []string{
		"invalid email", "invalid.com", "invalid@",
		s1 + "@aa", "@invalid", s2 + "@example.com",
		"@@", "a@@",
	}

	for _, x := range shouldFailAfterRev9 {
		t.Run(x, func(t *testing.T) {
			err := ValidateEmail(x, 9)
			assert.Error(t, err)
		})
	}
}

func TestBigInt2HexInt(t *testing.T) {
	values := []int64{-100, 0, 100}
	for _, value := range values {
		h := BigInt2HexInt(big.NewInt(value))
		assert.IsType(t, &common.HexInt{}, h)
		assert.Equal(t, value, h.Value().Int64())
	}
}

func TestMin(t *testing.T) {
	type arg struct {
		v0  int
		v1  int
		min int
	}

	args := []arg{
		{0, 0, 0},
		{1, 1, 1},
		{-1, 1, -1},
		{1, -1, -1},
		{1, 2, 1},
		{2, 1, 1},
	}

	for i, a := range args {
		name := fmt.Sprintf("test-%d", i)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, a.min, Min(a.v0, a.v1))
		})
	}
}

func TestIsNil(t *testing.T) {
	var addr *common.Address
	var i interface{} = addr

	assert.True(t, addr == nil)
	assert.False(t, i == nil)
	assert.True(t, IsNil(i))
}

func TestICXToIScore(t *testing.T) {
	type arg struct {
		icx    int64
		iscore int64
	}

	args := []arg{
		{0, 0},
		{1, 1000},
	}

	for i, a := range args {
		name := fmt.Sprintf("test-%d", i)
		t.Run(name, func(t *testing.T) {
			ret := ICXToIScore(big.NewInt(a.icx))
			assert.Equal(t, a.iscore, ret.Int64())
		})
	}
}

func TestIScoreToICX(t *testing.T) {
	type arg struct {
		iscore int64
		icx    int64
	}

	args := []arg{
		{0, 0},
		{1, 0},
		{10, 0},
		{100, 0},
		{1000, 1},
		{10000, 10},
	}

	for i, a := range args {
		name := fmt.Sprintf("test-%d", i)
		t.Run(name, func(t *testing.T) {
			ret := IScoreToICX(big.NewInt(a.iscore))
			assert.Equal(t, a.icx, ret.Int64())
		})
	}
}

func TestMergeMaps(t *testing.T) {
	m0 := map[string]interface{}{
		"a": 0,
		"b": 1,
	}

	m1 := map[string]interface{}{
		"b": 11,
		"c": 12,
	}

	m2 := map[string]interface{}{
		"d": 13,
		"e": 14,
	}

	ret := MergeMaps()
	assert.Nil(t, ret)

	ret = MergeMaps(m0)
	assert.True(t, reflect.DeepEqual(ret, m0))
	ret["c"] = 100
	_, ok := m0["c"]
	assert.False(t, ok)
	assert.False(t, reflect.DeepEqual(ret, m0))

	ret = MergeMaps(m0, m1)
	assert.Equal(t, 3, len(ret))
	assert.Equal(t, 0, ret["a"].(int))
	assert.Equal(t, 11, ret["b"].(int))
	assert.Equal(t, 12, ret["c"].(int))

	ret = MergeMaps(m0, m1, m2)
	assert.Equal(t, 5, len(ret))
	assert.Equal(t, 0, ret["a"].(int))
	assert.Equal(t, 11, ret["b"].(int))
	assert.Equal(t, 12, ret["c"].(int))
	assert.Equal(t, 13, ret["d"].(int))
	assert.Equal(t, 14, ret["e"].(int))

	assert.Equal(t, 2, len(m0))
	assert.Equal(t, 0, m0["a"].(int))
	assert.Equal(t, 1, m0["b"].(int))

	assert.Equal(t, 2, len(m1))
	assert.Equal(t, 11, m1["b"].(int))
	assert.Equal(t, 12, m1["c"].(int))

	assert.Equal(t, 2, len(m2))
	assert.Equal(t, 13, m2["d"].(int))
	assert.Equal(t, 14, m2["e"].(int))
}

func TestValidateCountryAlpha3(t *testing.T) {
	type arg struct {
		alpha3 string
		valid  bool
	}

	args := []arg{
		{"KOR", true},
		{"kor", true},
		{"Kor", true},
		{"kOr", true},
		{"koR", true},
		{"USA", true},
		{"usa", true},
		{"FRA", true},
		{"fra", true},
		{"JPN", true},
		{"jpn", true},
		{"CHN", true},
		{"chn", true},
		{"abc", false},
		{"000", false},
		{"k0r", false},
	}

	for _, a := range args {
		t.Run("test-"+a.alpha3, func(t *testing.T) {
			err := ValidateCountryAlpha3(a.alpha3)
			if a.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMinBigInt(t *testing.T) {
	args := []struct {
		v0, v1, min int64
	}{
		{0, 0, 0},
		{0, 100, 0},
		{100, 0, 0},
		{100, 200, 100},
		{100, 100, 100},
		{200, 100, 100},
		{-200, 0, -200},
		{0, -200, -200},
		{-100, -100, -100},
		{-100, -200, -200},
		{-200, -100, -200},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			v0 := big.NewInt(arg.v0)
			v1 := big.NewInt(arg.v1)
			min := big.NewInt(arg.min)
			assert.Equal(t, min.Int64(), MinBigInt(v0, v1).Int64())
		})
	}
}

func TestCalcPower(t *testing.T) {
	args := []struct {
		br     icmodule.Rate
		bonded int64
		voted  int64
		power  int64
	}{
		{0, 0, 0, 0},
		{0, 1000, 1000, 1000},
		{0, 1000, 3000, 3000},
		{500, 1000, 3000, 3000},
		{500, 100, 3000, 2000},
		{500, 0, 3000, 0},
		{10000, 100, 3000, 100},
		{10000, 0, 3000, 0},
		{10000, 10000, 20000, 10000},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			bonded := big.NewInt(arg.bonded)
			voted := big.NewInt(arg.voted)
			expPower := big.NewInt(arg.power)
			power := CalcPower(icmodule.Rate(arg.br), bonded, voted)
			assert.Zero(t, expPower.Cmp(power))
		})
	}
}

func TestMatchAll(t *testing.T) {
	args := []struct {
		flags   int
		flag    int
		success bool
	}{
		{1, 1, true},
		{2, 1, false},
		{3, 1, true},
		{3, 2, true},
		{1, 3, false},
		{2, 3, false},
		{3, 3, true},
		{0, 1, false},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arg.success, MatchAll(arg.flags, arg.flag))
		})
	}
}

func TestMatchAny(t *testing.T) {
	args := []struct {
		flags   int
		flag    int
		success bool
	}{
		{1, 1, true},
		{2, 1, false},
		{3, 1, true},
		{3, 2, true},
		{1, 3, true},
		{2, 3, true},
		{3, 3, true},
		{0, 3, false},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arg.success, MatchAny(arg.flags, arg.flag))
		})
	}
}
