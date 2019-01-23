package consensus

import (
	"io"
	"io/ioutil"
	"os"
)

const (
	configEnableDebugLog = false
)

var debugWriter io.Writer

func init() {
	if configEnableDebugLog {
		debugWriter = os.Stderr
	} else {
		debugWriter = ioutil.Discard
	}
}
