package fastsync

const (
	logLevelNone = iota
	logLevelMsg
	logLevelDebug
)

const configLogLevel = logLevelMsg

const (
	logMsg   = configLogLevel >= logLevelMsg
	logDebug = configLogLevel >= logLevelDebug
)
