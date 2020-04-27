package log

import (
	"fmt"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

type customFormatter struct{}

var levelNames = []string{"P", "F", "E", "W", "I", "D", "T"}

func (customFormatter) Format(e *logrus.Entry) ([]byte, error) {

	buf := e.Buffer
	fmt.Fprint(buf, levelNames[e.Level], "|")
	fmt.Fprint(buf, e.Time.Format(LogTimeLayout), "|")
	if v, ok := e.Data[FieldKeyWallet]; ok {
		buf.WriteString(fmt.Sprint(v, "----")[0:4])
		buf.WriteString("|")
	} else {
		buf.WriteString("----|")
	}
	if v, ok := e.Data[FieldKeyCID]; ok {
		fmt.Fprint(buf, v, "|")
	} else {
		buf.WriteString("-|")
	}
	if v, ok := e.Data[FieldKeyModule]; ok {
		fmt.Fprint(buf, v, "|")
	} else {
		if e.HasCaller() {
			fmt.Fprint(buf, getPackageName(e.Caller.Function), "|")
		} else {
			fmt.Fprint(buf, "--|")
		}
	}
	if v, ok := e.Data[FieldKeyEID]; ok {
		s := string(([]byte(fmt.Sprint(v)))[0:8])
		fmt.Fprint(buf, s, "|")
	}
	if v, ok := e.Data[FieldKeyPrefix]; ok {
		fmt.Fprint(buf, v)
	}
	if e.HasCaller() {
		fmt.Fprint(buf, path.Base(e.Caller.File), ":", e.Caller.Line, " ")
	}
	buf.WriteString(strings.TrimRight(e.Message, "\n"))
	for k, v := range e.Data {
		if _, ok := systemFields[k]; ok {
			continue
		}
		fmt.Fprintf(buf, " %s=%v", k, v)
	}
	buf.WriteString("\n")
	return buf.Bytes(), nil
}
