/**
errors provide error code
*/
package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodedError(t *testing.T) {
	msg := "First argument is incorrect"
	e := Errorc(IllegalArgumentError, msg)
	if c := CodeOf(e); c != IllegalArgumentError {
		t.Errorf("Expected = %d return = %d", IllegalArgumentError, c)
	}
}

func TestWrap(t *testing.T) {
	e := Errorc(IllegalArgumentError, "Method doesn't supported")
	e2 := Wrap(e, "Using Code Itself")

	if c := CodeOf(e2); c != IllegalArgumentError {
		t.Error("Wrapped error change error code to Unknown")
	}

	e3 := WithCode(e, UnsupportedError)
	if c := CodeOf(e3); c != UnsupportedError {
		t.Error("WithCode doesn't change error code")
	}
}

func TestWithCode(t *testing.T) {
	codes := []Code{
		IllegalArgumentError, UnsupportedError,
	}
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
	}{
		{"Errorc", args{
			Errorc(UnsupportedError, "Unsupported"),
		}},
		{"WithStack", args{
			WithStack(Errorc(UnsupportedError, "Unsupported")),
		}},
		{"WithCode", args{
			WithCode(Errorc(UnsupportedError, "Unsupported"), IllegalArgumentError),
		}},
		{"NewBase", args{
			NewBase(UnsupportedError, "Unsupported"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, c := range codes {
				e2 := WithCode(tt.args.err, c)
				if c2 := CodeOf(e2); c2 != c {
					t.Errorf("Returned code=%d exp=%d", c2, c)
				}
			}
		})
	}
	t.Run("Nil", func(t *testing.T) {
		err := WithCode(nil, IllegalArgumentError)
		assert.NoError(t, err)
	})
}

func TestErrorf(t *testing.T) {
	base := New("TestLog")
	e1 := fmt.Errorf("WrappingError err=%w", base)
	e2 := Errorf("WrappingError err=%w", base)

	assert.Equal(t, fmt.Sprintf("%s", e1), fmt.Sprintf("%s", e2))
	assert.Equal(t, fmt.Sprintf("%q", e1), fmt.Sprintf("%q", e2))
	assert.Equal(t, fmt.Sprintf("%v", e1), fmt.Sprintf("%v", e2))
	assert.Equal(t, Unwrap(e1), Unwrap(e2))

	t.Logf("fmt.Errorf() --> %+v", e1)
	t.Logf("Error(fmt.Errorf()) --> %+v", Error(e1))
	t.Logf("Errorf() --> %+v", e2)
}

func TestWithStack(t *testing.T) {
	e1 := New("TestLog")
	e2 := errors.New("TestLog")
	e3 := WithStack(e2)

	assert.Equal(t, fmt.Sprintf("%v", e2), fmt.Sprintf("%v", e3))
	assert.NotEqual(t, fmt.Sprintf("%+v", e2), fmt.Sprintf("%+v", e3))
	assert.Equal(t, e2, Unwrap(e3))
	assert.Nil(t, Unwrap(e1))
}

func TestFormat(t *testing.T) {
	type args struct {
		e error
	}
	tests := []struct {
		name  string
		args  args
		wantV string
	}{
		{"New()", args{New("BasicError")}, "BasicError"},
		{"codedError", args{NewBase(IllegalArgumentError, "IllegalArgumentTest")}, "E1002:IllegalArgumentTest"},
		{"withCodeAndStack", args{Errorc(IllegalArgumentError, "IllegalArgumentTest")}, "E1002:IllegalArgumentTest"},
		{"wrappedWithCode", args{WithCode(Errorc(IllegalArgumentError, "IllegalArgumentTest"), UnsupportedError)}, "E1003:IllegalArgumentTest"},
		{"wrappedWithMessage", args{Wrap(New("BasicError"), "TestError")}, "TestError"},
		{"wrappedWithCodeMessage", args{Wrapc(New("BasicError"), IllegalArgumentError, "TestError")}, "E1002:TestError"},
		{"wrappedWithStack#1", args{WithStack(Wrapc(New("BasicError"), IllegalArgumentError, "TestError"))}, "E1002:TestError"},
		{"wrappedWithStack#2", args{WithStack(NewBase(IllegalArgumentError, "IllegalArgument"))}, "E1002:IllegalArgument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.args.e.Error()
			s := fmt.Sprintf("%s", tt.args.e)
			assert.Equal(t, errMsg, s)

			q := fmt.Sprintf("%q", tt.args.e)
			assert.Equal(t, "\""+errMsg+"\"", q)

			v := fmt.Sprintf("%v", tt.args.e)
			assert.Equal(t, tt.wantV, v)

			vv := fmt.Sprintf("%+v", tt.args.e)
			assert.True(t, strings.Contains(vv, v))

			t.Logf("%%s:%s %%q:%s %%v:%s %%+v:%s", s, q, v, vv)
		})
	}
}

func TestWrapc(t *testing.T) {
	e := Errorc(UnsupportedError, "Method doesn't supported")
	e2 := Wrapc(e, IllegalArgumentError, "Argument is invalid")

	if c, ok := CoderOf(e2); !ok {
		t.Error("Fail to get Coder from Wrapc() output")
	} else {
		if c != e2 {
			t.Errorf("Returned error isn't expected one:%+v", c)
		}
	}
	if !Is(e2, e) {
		t.Errorf("Is(Wrapc(err), err) is TRUE")
	}
}

func TestNewBase(t *testing.T) {
	msg := "IllegalArgument"
	e := NewBase(IllegalArgumentError, msg)
	if c := CodeOf(e); c != IllegalArgumentError {
		t.Error("Code of NewBase() isn't CodeIllegalArgument")
	}
	assert.Equal(t, msg, e.Error())
	assert.Equal(t, msg, fmt.Sprintf("%s", e))
}

func TestCodeOf(t *testing.T) {
	base := NewBase(InvalidStateError, "InvalidState")
	type args struct {
		e error
	}
	tests := []struct {
		name string
		args args
		want Code
	}{
		{"Nil", args{nil}, Success},
		{"New1", args{New("Empty")}, UnknownError},
		{"New2", args{Errorf("Test(%d)", 1)}, UnknownError},
		{"New3", args{errors.New("MyError")}, UnknownError},
		{"NewBase1", args{NewBase(UnsupportedError, "MyError")}, UnsupportedError},
		{"NewBase2", args{WithStack(NewBase(UnsupportedError, "MyError"))}, UnsupportedError},
		{"Errorc", args{Errorc(CriticalHashError, "Supplied transaction is invalid")}, CriticalHashError},
		{"Wrapf", args{Wrapf(New("JSON Error"), "Supplied transaction is invalid")}, UnknownError},
		{"Wrapc1", args{Wrapc(New("JSON Error"), IllegalArgumentError, "Supplied transaction is invalid")}, IllegalArgumentError},
		{"Wrapcf1", args{Wrapcf(New("JSON Error"), IllegalArgumentError, "Supplied %s is invalid", "transaction")}, IllegalArgumentError},
		{"Errorc1", args{Errorc(IllegalArgumentError, "Supplied transaction is invalid")}, IllegalArgumentError},
		{"Errorcf1", args{Errorcf(UnsupportedError, "Feature(%d) isn't supported", 2)}, UnsupportedError},
		{"WithCode", args{WithCode(errors.New("SimpleError"), UnsupportedError)}, UnsupportedError},
		{"Code1", args{CriticalFormatError.AttachTo(errors.New("CriticalFormatTest"))}, CriticalFormatError},
		{"Code2", args{IllegalArgumentError.Wrap(errors.New("IllegalArgumentTest"), "Test")}, IllegalArgumentError},
		{"Code3", args{IllegalArgumentError.New("IllegalArgumentTest")}, IllegalArgumentError},
		{"Code4", args{IllegalArgumentError.Errorf("IllegalArgument%s", "Test")}, IllegalArgumentError},
		{"Code5", args{IllegalArgumentError.Wrapf(errors.New("IllegalArgumentTest"), "Test%s", "Error")}, IllegalArgumentError},
		{"Override1", args{Wrapc(base, IllegalArgumentError, "Wrapc")}, IllegalArgumentError},
		{"Override2", args{WithCode(base, IllegalArgumentError)}, IllegalArgumentError},
		{"Override3", args{IllegalArgumentError.Wrap(base, "Code.Wrap")}, IllegalArgumentError},
		{"Override4", args{IllegalArgumentError.AttachTo(base)}, IllegalArgumentError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeOf(tt.args.e); got != tt.want {
				t.Errorf("CodeOf() = %v, want %v", got, tt.want)
			}
			assert.Equal(t, IsCriticalCode(tt.want), IsCritical(tt.args.e))
			assert.True(t, tt.want.Equals(tt.args.e))
		})
	}
}

func TestIs(t *testing.T) {
	e := Errorc(IllegalArgumentError, "IllegalArgument")
	type args struct {
		e1, e2 error
	}
	cases := []struct {
		name string
		args args
		want bool
	}{
		{"WithNil1", args{e, nil}, false},
		{"WithNil2", args{nil, e}, false},
		{"NilNil", args{nil, nil}, true},
		{"Same", args{e, e}, true},
		{"Wrap", args{Wrap(e, "MyTest"), e}, true},
		{"Wrapc", args{Wrapc(e, UnsupportedError, "MyTest"), e}, true},
		{"Defined1", args{ErrIllegalArgument, e}, false},
		{"Defined2", args{e, ErrIllegalArgument}, false},
		{"fmt.Errorf", args{fmt.Errorf("error from %w", e), e}, true},
		{"WithStack", args{WithStack(e), e}, true},
		{"WithCode", args{WithCode(e, UnsupportedError), e}, true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Is(tt.args.e1, tt.args.e2))
			assert.Equal(t, tt.want, errors.Is(tt.args.e1, tt.args.e2))
			assert.Equal(t, errors.Is(tt.args.e2, tt.args.e1), Is(tt.args.e2, tt.args.e1))
		})
	}
}

func TestToString(t *testing.T) {
	assert.Equal(t, "", ToString(nil))

	e1 := IllegalArgumentError.New("Test")
	es1 := fmt.Sprintf("%v", e1)
	assert.Equal(t, es1, ToString(e1))
}

func TestWrappingWithNil(t *testing.T) {
	assert.NoError(t, WithStack(nil))
	assert.NoError(t, WithCode(nil, IllegalArgumentError))
	assert.NoError(t, Wrap(nil, "TestMessage"))
	assert.NoError(t, Wrapf(nil, "TestFormat(%d)", 10))
	assert.NoError(t, Wrapc(nil, IllegalArgumentError, "TestMessage"))
	assert.NoError(t, Wrapcf(nil, IllegalArgumentError, "TestFormat(%d)", 1))
	assert.NoError(t, IllegalArgumentError.AttachTo(nil))
	assert.NoError(t, IllegalArgumentError.Wrap(nil, "TestMessage"))
	assert.NoError(t, IllegalArgumentError.Wrapf(nil, "TestFormat(%d)", 10))
}
