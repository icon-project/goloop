/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lcstore

import (
	"encoding/json"
	"net/http"

	"github.com/icon-project/goloop/client"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/intconv"
)

type NodeDB struct {
	client *client.JsonRpcClient
}

type heightParam struct {
	Height common.HexInt64 `json:"height"`
}

type hashParam struct {
	Hash common.HexBytes `json:"hash"`
}

type txHashParam struct {
	TxHash common.HexBytes `json:"txHash"`
}

func (s *NodeDB) GetBlockJSONByHeight(height int) ([]byte, error) {
	result, err := s.client.Do("icx_getBlock", &heightParam{
		common.HexInt64{Value: int64(height)},
	}, nil)
	if err != nil {
		return nil, err
	} else {
		return result.Result, nil
	}
}

func (s *NodeDB) GetBlockJSONByID(id []byte) ([]byte, error) {
	result, err := s.client.Do("icx_getBlock", &hashParam{
		common.HexBytes(id),
	}, nil)
	if err != nil {
		return nil, err
	} else {
		return result.Result, nil
	}
}

func (s *NodeDB) GetLastBlockJSON() ([]byte, error) {
	result, err := s.client.Do("icx_getBlock", nil, nil)
	if err != nil {
		return nil, err
	} else {
		return result.Result, nil
	}
}

func (s *NodeDB) GetTransactionInfoJSONByTransaction(id []byte) ([]byte, error) {
	var receipt map[string]interface{}
	_, err := s.client.Do("icx_getTransactionResult", &txHashParam{
		common.HexBytes(id),
	}, &receipt)
	if err != nil {
		return nil, err
	}
	var tx map[string]interface{}
	_, err = s.client.Do("icx_getTransactionByHash", &txHashParam{
		common.HexBytes(id),
	}, &tx)
	if err != nil {
		return nil, err
	}
	info := make(map[string]interface{})
	info["block_hash"] = receipt["blockHash"]
	bh, _ := intconv.ParseInt(receipt["blockHeight"].(string), 64)
	info["block_height"] = bh
	info["tx_index"] = receipt["txIndex"]
	info["receipt"] = receipt
	info["transaction"] = tx
	delete(tx, "blockHash")
	delete(tx, "blockHeight")
	delete(tx, "txIndex")
	return json.Marshal(info)
}

func (s *NodeDB) GetRepsJSONByHash(id []byte) ([]byte, error) {
	result, err := s.client.Do(
		"rep_getListByHash",
		map[string]interface{}{
			"repsHash": common.HexBytes(id),
		},
		nil,
	)
	if err != nil {
		return nil, err
	}
	return result.Result, nil
}

func OpenNodeDB(endpoint string) (Database, error) {
	hc := new(http.Client)
	jc := client.NewJsonRpcClient(hc, endpoint)
	return &NodeDB{jc}, nil
}
