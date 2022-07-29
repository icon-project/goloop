package errors

import (
	"errors"
	"testing"
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
}

func TestFormat(t *testing.T) {
	e := Errorc(UnsupportedError, "Method doesn't supported")
	t.Log("Errorc(CodeUnsupported) -->")
	t.Logf("%+v", e)

	e2 := Wrapc(e, IllegalArgumentError, "Argument is invalid")
	t.Log("Wrapc(Errorc(CodeUnsupported),CodeIllegalArgument) -->")
	t.Logf("%+v", e2)

	e3 := WithCode(e2, UnknownError)
	t.Log("WithCode(CodeUnknown,Wrapc(Errorc(CodeUnsupported),CodeIllegalArgument)) -->")
	t.Logf("%+v", e3)

	e4 := WithCode(errors.New("SimpleError"), IllegalArgumentError)
	t.Log("WithCode(errors.New(),CodeIllegalArgument) -->")
	t.Logf("%+v", e4)
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
	e := NewBase(IllegalArgumentError, "IllegalArgument")
	if c := CodeOf(e); c != IllegalArgumentError {
		t.Error("Code of NewBase() isn't CodeIllegalArgument")
	}
}

func TestCodeOf(t *testing.T) {
	type args struct {
		e error
	}
	tests := []struct {
		name string
		args args
		want Code
	}{
		{"New1", args{New("Empty")}, UnknownError},
		{"New2", args{Errorf("Test(%d)", 1)}, UnknownError},
		{"New3", args{errors.New("MyError")}, UnknownError},
		{"NewBase1", args{NewBase(UnsupportedError, "MyError")}, UnsupportedError},
		{"NewBase2", args{WithStack(NewBase(UnsupportedError, "MyError"))}, UnsupportedError},
		{"Wrapc1", args{Wrapc(New("JSON Error"), IllegalArgumentError, "Supplied transaction is invalid")}, IllegalArgumentError},
		{"Errorc1", args{Errorc(IllegalArgumentError, "Supplied transaction is invalid")}, IllegalArgumentError},
		{"Errorcf1", args{Errorcf(UnsupportedError, "Feature(%d) isn't supported", 2)}, UnsupportedError},
		{"WithCode", args{WithCode(errors.New("SimpleError"), UnsupportedError)}, UnsupportedError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeOf(tt.args.e); got != tt.want {
				t.Errorf("CodeOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIs(t *testing.T) {
	e := Errorc(IllegalArgumentError, "IllegalArgument")

	e2 := Wrap(e, "MyTest")
	if Is(e, e2) || errors.Is(e, e2) {
		t.Error("Fail to check !Is(origin, Wrap(origin)) is FALSE")
	}
	if !Is(e2, e) || !errors.Is(e2, e) {
		t.Error("Fail to check Is(Wrap(origin), origin) is TRUE")
	}

	e3 := Wrapc(e, UnsupportedError, "MyTest2")
	if Is(e, e3) || errors.Is(e, e3) {
		t.Error("Fail to check !Is(origin, Wrapc(origin)) is FALSE")
	}
	if !Is(e3, e) || !errors.Is(e3, e) {
		t.Error("Fail to check Is(Wrapc(origin), origin) is TRUE")
	}
}
