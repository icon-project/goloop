package contract

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/state"
)

type SystemScore interface {
	GetAPI() *scoreapi.Info
	Invoke(method string, paramObj *codec.TypedObj) (module.Status, *codec.TypedObj)
}

func GetSystemScore(from, to module.Address, cc CallContext) SystemScore {
	// chain score
	// addOn score - static, dynamic
	if bytes.Equal(to.ID(), state.SystemID) == true {
		return &ChainScore{from, to, cc}
	}
	// get account for to
	// get & load so
	// get instance for it.
	return nil
}
