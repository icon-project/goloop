package jsonrpc

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {

	validator := NewValidator()

	var param struct {
		Hash    HexBytes `json:"hash" validate:"required,t_hash"`
		Height  HexInt   `json:"height" validate:"optional,t_int"`
		Address Address  `json:"address" validate:"required,t_addr"`
	}

	params := []byte(`
		{
			"hash": "0xb5f908339f447ca97525a3eb8c3e450e767ffe3e242df3f87e4af4295e1277f3",
			"height": "0x10",
			"address": "cx94b475b51924f4a2f449b982e5bfa1a47055a66f"
		}
	`)

	if err := json.Unmarshal(params, &param); err != nil {
		assert.Fail(t, "unmarshal fail", err.Error())
	}
	if err := validator.Validate(&param); err != nil {
		assert.Fail(t, "validate fail", err.Error())
	}

	assert.Equal(t, "b5f908339f447ca97525a3eb8c3e450e767ffe3e242df3f87e4af4295e1277f3", hex.EncodeToString(param.Hash.Bytes()))
	assert.Equal(t, int64(0x10), param.Height.Value())
	assert.Equal(t, "cx94b475b51924f4a2f449b982e5bfa1a47055a66f", param.Address.Address().String())

}
