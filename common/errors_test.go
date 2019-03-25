package common

import (
	"fmt"
	"testing"
)

func TestBaseError_WithStack(t *testing.T) {
	e := ErrIllegalArgument.WithStack()
	if c := ErrorCodeOf(e); c != ErrorCodeIllegalArgument {
		t.Errorf("Returned code=%d expected=%d", c, ErrorCodeIllegalArgument)
	}
	fmt.Printf("BaseError.WithStack():%+v\n------------------------------------\n", e)

	e = ErrorCodeIllegalArgument.Error("arg2 argument is illegal")
	if c := ErrorCodeOf(e); c != ErrorCodeIllegalArgument {
		t.Errorf("Returned code=%d expected=%d", c, ErrorCodeIllegalArgument)
	}
	fmt.Printf("ErrorCode.Error():%+v\n------------------------------------\n", e)

	e = Error(ErrorCodeIllegalArgument, "myTest")
	if c := ErrorCodeOf(e); c != ErrorCodeIllegalArgument {
		t.Errorf("Returned code=%d expected=%d", c, ErrorCodeIllegalArgument)
	}
	fmt.Printf("Error():%+v\n------------------------------------\n", e)

	e = Error(ErrorCodeUnknown, "myTest1")
	e = Wrap(ErrorCodeIllegalArgument, e)
	if c := ErrorCodeOf(e); c != ErrorCodeIllegalArgument {
		t.Errorf("Returned code=%d expected=%d", c, ErrorCodeIllegalArgument)
	}
	fmt.Printf("Wrap():%+v\n------------------------------------\n", e)
}
