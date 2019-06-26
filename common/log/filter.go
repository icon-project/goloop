package log

import "github.com/sirupsen/logrus"

type logFilter struct {
	formatter    logrus.Formatter
	defaultLevel Level
	moduleLevels map[string]Level
}

func newLogFilter(formatter logrus.Formatter) *logFilter {
	return &logFilter{
		formatter:    formatter,
		defaultLevel: TraceLevel,
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

	if e.Level > logrus.Level(level) {
		return nil, nil
	}
	return f.formatter.Format(e)
}

func (f *logFilter) SetModuleLevel(module string, level Level) {
	f.moduleLevels[module] = level
}

func (f *logFilter) SetDefaultLevel(level Level) {
	f.defaultLevel = level
}
