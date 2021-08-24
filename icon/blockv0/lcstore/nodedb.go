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
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/icon-project/goloop/client"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/server/jsonrpc"
)

type NodeDB struct {
	client *client.JsonRpcClient
	tr     *tpsRegulator
}

type heightParam struct {
	Height      common.HexInt64 `json:"height"`
	Unconfirmed bool            `json:"unconfirmed,omitempty"`
}

type hashParam struct {
	Hash common.HexBytes `json:"hash"`
}

type txHashParam struct {
	TxHash common.HexBytes `json:"txHash"`
}

func isJSONRpcError(err error, code jsonrpc.ErrorCode, msg string) bool {
	if rpcErr, ok := err.(*jsonrpc.Error); ok {
		return rpcErr.Code == code && (len(msg) == 0 || rpcErr.Message == msg)
	}
	return false
}

func (s *NodeDB) GetBlockJSONByHeight(height int, unconfirmed bool) ([]byte, error) {
	s.tr.Wait()
	result, err := s.Do("icx_getBlock", &heightParam{
		common.HexInt64{Value: int64(height)},
		unconfirmed,
	}, nil)
	if err != nil {
		if isJSONRpcError(err, jsonrpc.ErrorCodeInvalidParams, "fail wrong block height") {
			return nil, errors.ErrNotFound
		}
		return nil, err
	} else {
		return result.Result, nil
	}
}

func (s *NodeDB) GetBlockJSONByID(id []byte) ([]byte, error) {
	s.tr.Wait()
	result, err := s.Do("icx_getBlock", &hashParam{
		common.HexBytes(id),
	}, nil)
	if err != nil {
		if isJSONRpcError(err, jsonrpc.ErrorCodeInvalidParams, "fail wrong block hash") {
			return nil, errors.ErrNotFound
		}
		return nil, err
	} else {
		return result.Result, nil
	}
}

func (s *NodeDB) GetLastBlockJSON() ([]byte, error) {
	s.tr.Wait()
	result, err := s.Do("icx_getBlock", nil, nil)
	if err != nil {
		return nil, err
	} else {
		return result.Result, nil
	}
}

func (s *NodeDB) GetTransactionJSON(id []byte) ([]byte, error) {
	var tx json.RawMessage
	_, err := s.Do("icx_getTransactionByHash", &txHashParam{
		common.HexBytes(id),
	}, &tx)
	if err != nil {
		if isJSONRpcError(err, jsonrpc.ErrorCodeInvalidParams, "Invalid params txHash") {
			return nil, errors.ErrNotFound
		}
		return nil, err
	} else {
		return tx, nil
	}
}

func (s *NodeDB) Do(method string, param interface{}, res interface{}) (*client.Response, error) {
	s.tr.Wait()
	return s.client.Do(method, param, res)
}

func (s *NodeDB) GetResultJSON(id []byte) ([]byte, error) {
	var receipt map[string]interface{}
	_, err := s.Do("icx_getTransactionResult", &txHashParam{
		common.HexBytes(id),
	}, &receipt)
	if err != nil {
		if isJSONRpcError(err, jsonrpc.ErrorCodeInvalidParams, "Invalid params txHash") {
			return nil, errors.ErrNotFound
		}
		return nil, err
	}
	var tx map[string]interface{}
	_, err = s.Do("icx_getTransactionByHash", &txHashParam{
		common.HexBytes(id),
	}, &tx)
	if err != nil {
		if isJSONRpcError(err, jsonrpc.ErrorCodeInvalidParams, "Invalid params txHash") {
			return nil, errors.ErrNotFound
		}
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

func (s *NodeDB) GetReceiptJSON(id []byte) ([]byte, error) {
	var receipt json.RawMessage
	_, err := s.Do("icx_getTransactionResult", &txHashParam{
		common.HexBytes(id),
	}, &receipt)
	if err != nil {
		if isJSONRpcError(err, jsonrpc.ErrorCodeInvalidParams, "Invalid params txHash") {
			return nil, errors.ErrNotFound
		}
		return nil, err
	} else {
		return receipt, nil
	}
}

func (s *NodeDB) GetRepsJSONByHash(id []byte) ([]byte, error) {
	result, err := s.Do(
		"rep_getListByHash",
		map[string]interface{}{
			"repsHash": common.HexBytes(id),
		},
		nil,
	)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(result.Result, []byte("[]")) {
		return nil, errors.ErrNotFound
	}
	return result.Result, nil
}

func (s *NodeDB) Close() error {
	return nil
}

func (s *NodeDB) GetTPS() float32 {
	return s.tr.GetTPS()
}

func OpenNodeDB(endpoint string, rps int) (Database, error) {
	hc := new(http.Client)
	jc := client.NewJsonRpcClient(hc, endpoint)
	tr := new(tpsRegulator).Init(rps)
	return &NodeDB{jc, tr}, nil
}
