package v3

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/server/jsonrpc"
)

func TestCallParamValidator(t *testing.T) {

	validator := jsonrpc.NewValidator()
	RegisterValidationRule(validator)

	var callParam CallParam

	callParams := []byte(`
		{
            "from": "hx4873b94352c8c1f3b2f09aaeccea31ce9e90bd31",
            "to": "cx059e19601bcb1424884f4ef19addc0a03de9e9cd",
            "dataType": "call",
            "data": {
                "method": "balanceOf",
                "params": {
                    "_owner": "hx4e436ed6adf72b6d2a80613cc15d5af5ddb6701e"
                }
            }
		}
	`)

	if err := json.Unmarshal(callParams, &callParam); err != nil {
		// fmt.Printf("unmarshal error: %s\n", err.Error())
		assert.Fail(t, "unmarshal fail", err.Error())
	}

	bs, _ := json.MarshalIndent(&callParam, "", "\t")
	fmt.Println(string(bs))

	if err := validator.Validate(&callParam); err != nil {
		assert.Fail(t, "validate fail", err.Error())
	}

	var invalidCallParam CallParam
	invalidCallParams := []byte(`
		{
			"to": "cx0000000000000000000000000000000000000000",
			"dataType": "call",
			"data": {
				"method": "balanceOf",
				"params": 2
			}
		}
	`)
	if err := json.Unmarshal(invalidCallParams, &invalidCallParam); err != nil {
		// fmt.Printf("unmarshal error: %s\n", err.Error())
		assert.Fail(t, "unmarshal fail", err.Error())
	}

	bs, _ = json.MarshalIndent(&invalidCallParam, "", "\t")
	fmt.Println(string(bs))

	assert.Error(t, validator.Validate(&invalidCallParam))
}

func TestTransactionParamValidator(t *testing.T) {

	validator := jsonrpc.NewValidator()
	RegisterValidationRule(validator)

	var txParam TransactionParam

	txParams := []byte(`
		{
			"version": "0x3",
			"from": "hx4873b94352c8c1f3b2f09aaeccea31ce9e90bd31",
			"to": "cx059e19601bcb1424884f4ef19addc0a03de9e9cd",
			"value": "0x11",
			"stepLimit": "0x12345",
			"timestamp": "0x563a6cf330136",
			"nid": "0x3",
			"nonce": "0x1",
			"signature": "VAia7YZ2Ji6igKWzjR2YsGa2m53nKPrfK7uXYW78QLE+ATehAVZPC40szvAiA6NEU5gCYB4c4qaQzqDh2ugcHgA=",
			"dataType": "deploy",
			"data": {
                "contentType": "application/zip",
                "content": "0x121212",
                "params": {
                    "name": "ABCToken",
                    "symbol": "abc",
                    "decimals": "12"
				}
            }			
		}
	`)

	if err := json.Unmarshal(txParams, &txParam); err != nil {
		assert.Fail(t, "unmarshal fail", err.Error())
	}

	bs, _ := json.MarshalIndent(&txParam, "", "\t")
	fmt.Println(string(bs))

	if err := validator.Validate(&txParam); err != nil {
		assert.Fail(t, "validate fail", err.Error())
	}
}
