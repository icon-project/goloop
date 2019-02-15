package service

const (
	LogLevelNone = iota
	LogLevelMsg
	LogLevelDebug
)

const configLogLevel = LogLevelMsg

const (
	logMsg   = configLogLevel >= LogLevelMsg
	logDebug = configLogLevel >= LogLevelDebug
)
