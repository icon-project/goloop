package icutils

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
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
