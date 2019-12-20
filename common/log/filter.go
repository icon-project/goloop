package log

import (
	"io"

	"github.com/sirupsen/logrus"
)

type logFilter struct {
	formatter    logrus.Formatter
	defaultLevel Level
	moduleLevels map[string]Level

	fileWriter  io.Writer
	filterLevel Level
}

func newLogFilter(formatter logrus.Formatter) *logFilter {
	return &logFilter{
		formatter:    formatter,
		defaultLevel: TraceLevel,
		filterLevel:  TraceLevel,
		moduleLevels: make(map[string]Level, 6),
	}
}

func (f *logFilter) Format(e *logrus.Entry) ([]byte, error) {
	level := f.defaultLevel

	var module string
	if value, ok := e.Data[FieldKeyModule]; !ok {
		if e.HasCaller() {
			module = getPackageName(e.Caller.Function)
		}
	} else {
		module = value.(string)
	}

	if len(module) > 0 {
		if lv, ok := f.moduleLevels[module]; ok {
			level = lv
		}
	}

	if e.Level > logrus.Level(level) && f.fileWriter == nil {
		return nil, nil
	}
	buf, err := f.formatter.Format(e)
	if f.fileWriter != nil && len(buf) > 0 {
		f.fileWriter.Write(buf)
	}
	if e.Level > logrus.Level(level) {
		return nil, nil
	}
	return buf, err
}

func (f *logFilter) SetModuleLevel(module string, level Level) {
	f.moduleLevels[module] = level
}

func (f *logFilter) GetModuleLevel(module string) Level {
	if lv, ok := f.moduleLevels[module]; ok {
		return lv
	} else {
		return f.defaultLevel
	}
}

func (f *logFilter) SetDefaultLevel(level Level) {
	f.defaultLevel = level
}

func (f *logFilter) GetDefaultLevel() Level {
	return f.defaultLevel
}

// SetFileWriter set file writer
func (f *logFilter) SetFileWriter(writer io.Writer) error {
	f.fileWriter = writer
	return nil
}
