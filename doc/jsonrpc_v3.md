---
title: JSON-RPC v3
---

# Goloop JSON-RPC API v3

## Introduction

This document explains JSON-RPC APIs (version 3) available to interact with Goloop nodes.

The API end point is `http://<host>:<port>/api/v3/<channel>`

If there is one channel or there is a default channel then you may skip channel name. Channel name of the chain will be set on configuring the channel. It may use hexadecimal string of NID if it's not specified (ex: `a34` for 0xa34). For ICON networks, they uses `icon_dex` as channel name.

## Value Types

Basically, every VALUE in JSON-RPC message is string.
Below table shows the most common "VALUE types".

| VALUE type                            | Description                                       | Example                                                                                  |
|:--------------------------------------|:--------------------------------------------------|:-----------------------------------------------------------------------------------------|
| <a id="T_ADDR_EOA">T_ADDR_EOA</a>     | "hx" + 40 digit HEX string                        | hxbe258ceb872e08851f1f59694dac2558708ece11                                               |
| <a id="T_ADDR_SCORE">T_ADDR_SCORE</a> | "cx" + 40 digit HEX string                        | cxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32                                               |
| <a id="T_HASH">T_HASH</a>             | "0x" + 64 digit HEX string                        | 0xc71303ef8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238                       |
| <a id="T_INT">T_INT</a>               | "0x" + lowercase HEX string. No zero padding.     | 0xa                                                                                      |
| <a id="T_BOOL">T_BOOL</a>             | "0x1" for 'true', "0x0" for 'false'               | 0x1                                                                                      |
| <a id="T_BIN_DATA">T_BIN_DATA</a>     | "0x" + lowercase HEX string. Length must be even. | 0x34b2                                                                                   |
| <a id="T_SIG">T_SIG</a>               | base64 encoded string                             | VAia7YZ2Ji6igKWzjR2YsGa2m53nKPrfK7uXYW78QLE+ATehAVZPC40szvAiA6NEU5gCYB4c4qaQzqDh2ugcHgA= |
| <a id="T_DATA_TYPE">T_DATA_TYPE</a>   | Type of data                                      | call, deploy or message                                                                  |
| <a id="T_STRING">T_STRING</a>         | normal string                                     | test, hello, ...                                                                         |

## Failure Code

Following is a list of failure codes.

| Name                      | Value      | Description                                                                 |
|:--------------------------|:----------:|:----------------------------------------------------------------------------|
| UNKNOWN_FAILURE           | 1          | An uncategorized internal system error occurred.                            |
| CONTRACT_NOT_FOUND        | 2          | There is no valid contract on the target address.                           |
| METHOD_NOT_FOUND          | 3          | The specified method does not exist or is not usable.                       |
| METHOD_NOT_PAYABLE        | 4          | The specified method is not payable.                                        |
| ILLEGAL_FORMAT            | 5          | An Illegal method parameter or decorator has been declared.                 |
| INVALID_PARAMETER         | 6          | An invalid parameter has been passed to a method.                           |
| INVALID_INSTANCE          | 7          | An object has not been derived from the appropriate base class.             |
| INVALID_CONTAINER_ACCESS  | 8          | Invalid container access occurred.                                          |
| ACCESS_DENIED             | 9          | Access operation is denied, typically due to a database permission check.   |
| OUT_OF_STEP               | 10         | Out of step                                                                 |
| OUT_OF_BALANCE            | 11         | Out of balance                                                              |
| TIMEOUT_ERROR             | 12         | Timeout error                                                               |
| STACK_OVERFLOW            | 13         | Too deep inter-call                                                         |
| SKIP_TRANSACTION          | 14         | The transaction is not executed.                                            |
| REVERTED                  | 32 ~ 999   | End with revert request.(by Revision5, it was limited to 99)                |

## JSON-RPC Failure

> Failure object example
```json
{
  "code" : -32700,
  "message": "Parse error"
}
```

> Timeout object example
```json
{
  "code" : -31006,
  "message": "Timeout",
  "data": "0x402b630c5ed80d1b8f0d89ca14a091084bcc0f6a98bc52329bccc045415bc0bd"
}
```

`icx_sendTransactionAndWait` and `icx_waitTransactionResult` may return one of timeout errors.
In those cases, it would have transaction hash in `data` field.


#### Error Codes

Below table shows the default error messages for the error code. Actual message may vary depending on the implementation.

| Category     | Error code      | Message          | Description                                                                                               |
|:-------------|:----------------|:-----------------|:----------------------------------------------------------------------------------------------------------|
| Json Parsing | -32700          | Parse error      | Invalid JSON was received by the server.<br/>An error occurred on the server while parsing the JSON text. |
| RPC Parsing  | -32600          | Invalid Request  | The JSON sent is not a valid Request object.                                                              |
|              | -32601          | Method not found | The method does not exist / is not available.                                                             |
|              | -32602          | Invalid params   | Invalid method parameter(s).                                                                              |
|              | -32603          | Internal error   | Internal JSON-RPC error.                                                                                  |
| Server Error | -32000 ~ -32099 |                  | Server error.                                                                                             |
| System Error | -31000          | System Error     | Unknown system error.                                                                                     |
|              | -31001          | Pool Overflow    | Transaction pool overflow.                                                                                |
|              | -31002          | Pending          | Transaction is in the pool, but not included in the block.                                                |
|              | -31003          | Executing        | Transaction is included in the block, but it doesn’t have confirmed result.                               |
|              | -31004          | Not found        | Requested data is not found.                                                                              |
|              | -31005          | Lack of resource | Resource is not available.                                                                                |
|              | -31006          | Timeout          | Fail to get result of transaction in specified timeout                                                    |
|              | -31007          | System timeout   | Fail to get result of transaction in system timeout (short time than specified)                           |
| SCORE Error  | -30000 ~ -30999 |                  | Mapped errors from [Failure code](#failure-code) ( = -30000 - `value` )                                   |


## JSON-RPC HTTP Header

You may set HTTP header for extension data of the request.

**HTTP Header name** : `Icon-Options`


| Option       | Description                          | Allowed APIs |
|:-------------|:-------------------------------------|:-------------|
| timeout      | Timeout for waiting in millisecond   | icx_sendTransactionAndWait <br/> icx_waitTransactionResult |




## JSON-RPC Methods

### icx_getLastBlock

Returns the last block information.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getLastBlock",
}
```
#### Parameters

None

> Example responses

```json
{
  "jsonrpc": "2.0",
  "result": {
    "block_hash": "8e25acc5b5c74375079d51828760821fc6f54283656620b1d5a715edcc0770c6",
     "confirmed_transaction_list": [
      {
        "from": "hx84f6c686fba03bc7ca65d15ae844ee56ff24a32b",
        "nid": "0x1",
        "signature": "tCUwOb6vsaUKy+NYvmzdJYC0jm3Erd5cR6wKnVuAjzMOECC+t/oK7fG/Tz2Y3C25o0AfCmbneXpias6xco+43wE=",
        "stepLimit": "0x3e8",
        "timestamp": "0x58a14bfe9b904",
        "to": "hx244deea00413d85c6637e7fdd53afa697f29d08f",
        "txHash": "0xd8da71e926052b960def61c64f325412772f8e986f888685bc87c0bc046c2d9f",
        "value": "0xa",
        "version": "0x3"
      }
     ],
    "height": 512,
    "merkle_tree_root_hash": "5c8d4e59ded657c6acbb67030929dfcaf114a268d6d58df53e7174e40db74158",
    "peer_id": "hx4208599c8f58fed475db747504a80a311a3af63b",
    "prev_block_hash": "0fdf04d13229482e3533948d4582344a3d44c399e71ab12c653ae57bcbee5d90",
    "signature": "",
    "time_stamp": 1559204699330360,
    "version": "2.0"
  },
  "id": "1001"
}
```
#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

### icx_getBlockByHeight

Returns block information by block height.

> Request

```json
{
  "id": "1001",
  "jsonrpc": "2.0",
  "method": "icx_getBlockByHeight",
  "params": {
    "height": "0x100"
  }
}
```
#### Parameters

| KEY    | VALUE type      | Description               |
|:-------|:----------------|:--------------------------|
| height | [T_INT](#T_INT) | Integer of a block height |

> Example responses

```json
{
  "jsonrpc": "2.0",
  "result": {
    "block_hash": "8e25acc5b5c74375079d51828760821fc6f54283656620b1d5a715edcc0770c6",
     "confirmed_transaction_list": [
      {
        "from": "hx84f6c686fba03bc7ca65d15ae844ee56ff24a32b",
        "nid": "0x1",
        "signature": "tCUwOb6vsaUKy+NYvmzdJYC0jm3Erd5cR6wKnVuAjzMOECC+t/oK7fG/Tz2Y3C25o0AfCmbneXpias6xco+43wE=",
        "stepLimit": "0x3e8",
        "timestamp": "0x58a14bfe9b904",
        "to": "hx244deea00413d85c6637e7fdd53afa697f29d08f",
        "txHash": "0xd8da71e926052b960def61c64f325412772f8e986f888685bc87c0bc046c2d9f",
        "value": "0xa",
        "version": "0x3"
      }
     ],
    "height": 512,
    "merkle_tree_root_hash": "5c8d4e59ded657c6acbb67030929dfcaf114a268d6d58df53e7174e40db74158",
    "peer_id": "hx4208599c8f58fed475db747504a80a311a3af63b",
    "prev_block_hash": "0fdf04d13229482e3533948d4582344a3d44c399e71ab12c653ae57bcbee5d90",
    "signature": "",
    "time_stamp": 1559204699330360,
    "version": "2.0"
  },
  "id": "1001"
}
```

#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

### icx_getBlockByHash

Returns block information by block hash.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getBlockByHash",
  "params": {
      "hash": "8e25acc5b5c74375079d51828760821fc6f54283656620b1d5a715edcc0770c6"
  }
}
```
#### Parameters

| KEY  | VALUE type        | Description     |
|:-----|:------------------|:----------------|
| hash | [T_HASH](#T_HASH) | Hash of a block |

> Example responses

```json
{
  "jsonrpc": "2.0",
  "result": {
    "block_hash": "8e25acc5b5c74375079d51828760821fc6f54283656620b1d5a715edcc0770c6",
     "confirmed_transaction_list": [
      {
        "from": "hx84f6c686fba03bc7ca65d15ae844ee56ff24a32b",
        "nid": "0x1",
        "signature": "tCUwOb6vsaUKy+NYvmzdJYC0jm3Erd5cR6wKnVuAjzMOECC+t/oK7fG/Tz2Y3C25o0AfCmbneXpias6xco+43wE=",
        "stepLimit": "0x3e8",
        "timestamp": "0x58a14bfe9b904",
        "to": "hx244deea00413d85c6637e7fdd53afa697f29d08f",
        "txHash": "0xd8da71e926052b960def61c64f325412772f8e986f888685bc87c0bc046c2d9f",
        "value": "0xa",
        "version": "0x3"
      }
     ],
    "height": 512,
    "merkle_tree_root_hash": "5c8d4e59ded657c6acbb67030929dfcaf114a268d6d58df53e7174e40db74158",
    "peer_id": "hx4208599c8f58fed475db747504a80a311a3af63b",
    "prev_block_hash": "0fdf04d13229482e3533948d4582344a3d44c399e71ab12c653ae57bcbee5d90",
    "signature": "",
    "time_stamp": 1559204699330360,
    "version": "2.0"
  },
  "id": "1001"
}
```

#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

### icx_call

Calls SCORE's external function.

Does not make state transition (i.e., read-only).

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_call",
  "params": {
        "from": "hxbe258ceb872e08851f1f59694dac2558708ece11", // TX sender address
        "to": "cxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32",   // SCORE address
        "dataType": "call",
        "data": {
            "method": "get_balance", // SCORE external function
            "params": {
                "address": "hx1f9a3310f60a03934b917509c86442db703cbd52" // input parameter of "get_balance"
            }
        }
    }
}
```
#### Parameters

| KEY         | VALUE type                    | Required | Description                                    |
|:------------|:------------------------------|:---------|:-----------------------------------------------|
| from        | [T_ADDR_EOA](#T_ADDR_EOA)     | required | Message sender's address.                      |
| to          | [T_ADDR_SCORE](#T_ADDR_SCORE) | required | SCORE address that will handle the message.    |
| height      | [T_INT](#T_INT)               | optional | Integer of a block height                      |
| dataType    | [T_DATA_TYPE](#T_DATA_TYPE)   | required | `call` is the only possible data type.         |
| data        | JSON object                   | required | See [Parameters - data](#sendtxparameterdata). |
| data.method | JSON string                   | required | Name of the function.                          |
| data.params | JSON object                   | required | Parameters to be passed to the function.       |

> Example responses

```json
{
  "jsonrpc": "2.0",
  "result": "0x2961fff8ca4a62327800000",
  "id": 1001
}
```

#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success             ||

### icx_getBalance

Returns the ICX balance of the given EOA or SCORE.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getBalance",
   "params": {
        "address": "hxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32"
    }
}
```
#### Parameters

| KEY     | VALUE type                                                 | Required | Description               |
|:--------|:-----------------------------------------------------------|:---------|:--------------------------|
| address | [T_ADDR_EOA](#T_ADDR_EOA) or [T_ADDR_SCORE](#T_ADDR_SCORE) | required | Address of EOA or SCORE   |
| height  | [T_INT](#T_INT)                                            | optional | Integer of a block height |

> Example responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": "0xde0b6b3a7640000"
}
```
#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success             ||

### icx_getScoreApi

Returns SCORE's external API list.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getScoreApi",
  "params": {
      "address": "cxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32"  // SCORE address
  }
}
```
#### Parameters

| KEY     | VALUE type                    | Required | Description                   |
|:--------|:------------------------------|:---------|:------------------------------|
| address | [T_ADDR_SCORE](#T_ADDR_SCORE) | required | SCORE address to be examined. |
| height  | [T_INT](#T_INT)               | optional | Integer of a block height     |

> Example responses

```json
{
    "jsonrpc": "2.0",
    "id": 1234,
    "result": [
        {
            "type": "function",
            "name": "balanceOf",
            "inputs": [
                {
                    "name": "_owner",
                    "type": "Address"
                }
            ],
            "outputs": [
                {
                    "type": "int"
                }
            ],
            "readonly": "0x1"
        },
        {
            "type": "eventlog",
            "name": "FundTransfer",
            "inputs": [
                {
                    "name": "backer",
                    "type": "Address",
                    "indexed": "0x1"
                },
                {
                    "name": "amount",
                    "type": "int",
                    "indexed": "0x1"
                },
                {
                    "name": "is_contribution",
                    "type": "bool",
                    "indexed": "0x1"
                }
            ]
        },
        {...}
    ]
}
```
#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success             ||

* Fields containing information about the function
    - type : `function`, `fallback`, or `eventlog`
    - name : function name
    - inputs : parameters in array
        + name : parameter name
        + type : parameter type (`int`, `str`, `bytes`, `bool`, `Address`)
        + default: the default value if the parameter has a default value (optional)
        + indexed : `0x1` if the parameter is indexed (when this is `eventlog`) (optional)
    - outputs : return value
        + type : return value type (`int`, `str`, `bytes`, `bool`, `Address`, `dict`, `list`)
    - readonly : `0x1` if this is declared as `external(readonly=True)`
    - payable : `0x1` if this has `payable` decorator

### icx_getTotalSupply

Returns total ICX coin supply that has been issued.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getTotalSupply"
}
```
#### Parameters

| KEY     | VALUE type      | Required | Description               |
|:--------|:----------------|:---------|:--------------------------|
| height  | [T_INT](#T_INT) | optional | Integer of a block height |

> Example responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": "0x2961fff8ca4a62327800000"
}
```

#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success             ||

### icx_getTransactionResult

Returns the transaction result requested by transaction hash.

> Request

```json
{
  "jsonrpc": "2.0",
  "id": "1001",
  "method": "icx_getTransactionResult",
  "params": {
    "txHash": "0xd8da71e926052b960def61c64f325412772f8e986f888685bc87c0bc046c2d9f"
  }
}
```
#### Parameters

| KEY    | VALUE type        | Description             |
|:-------|:------------------|:------------------------|
| txHash | [T_HASH](#T_HASH) | Hash of the transaction |

> Example responses

```json
{
  "jsonrpc": "2.0",
  "result": {
    "blockHash": "0x8ef3b2a67262b9b1fe4b598059774472e9ccef401734335d87a4ba998cfd40fb",
    "blockHeight": "0x200",
    "cumulativeStepUsed": "0x0",
    "eventLogs": [],
    "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "status": "0x1",
    "stepPrice": "0x0",
    "stepUsed": "0x0",
    "to": "hx244deea00413d85c6637e7fdd53afa697f29d08f",
    "txHash": "0xd8da71e926052b960def61c64f325412772f8e986f888685bc87c0bc046c2d9f",
    "txIndex": "0x0"
  },
  "id": "1001"
}
```
#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

<a id="T_RESULT">Transaction Result</a>

| KEY                | VALUE type                                                 | Description                                                                            |
|:-------------------|:-----------------------------------------------------------|:---------------------------------------------------------------------------------------|
| status             | [T_INT](#T_INT)                                            | 1 on success, 0 on failure.                                                            |
| to                 | [T_ADDR_EOA](#T_ADDR_EOA) or [T_ADDR_SCORE](#T_ADDR_SCORE) | Recipient address of the transaction                                                   |
| failure            | JSON object                                                | This field exists when status is 0. Please refer [failure object](#T_FAILURE)          |
| txHash             | [T_HASH](#T_HASH)                                          | Transaction hash                                                                       |
| txIndex            | [T_INT](#T_INT)                                            | Transaction index in the block                                                         |
| blockHeight        | [T_INT](#T_INT)                                            | Height of the block that includes the transaction.                                     |
| blockHash          | [T_HASH](#T_HASH)                                          | Hash of the block that includes the transaction.                                       |
| cumulativeStepUsed | [T_INT](#T_INT)                                            | Sum of stepUsed by this transaction and all preceding transactions in the same block.  |
| stepUsed           | [T_INT](#T_INT)                                            | The amount of step used by this transaction.                                           |
| stepPrice          | [T_INT](#T_INT)                                            | The step price used by this transaction.                                               |
| scoreAddress       | [T_ADDR_SCORE](#T_ADDR_SCORE)                              | SCORE address if the transaction created a new SCORE. (optional)                       |
| eventLogs          | [T_ARRAY](#T_ARRAY)                                        | Array of eventlogs, which this transaction generated.                                  |
| logsBloom          | [T_BIN_DATA](#T_BIN_DATA)                                  | Bloom filter to quickly retrieve related eventlogs.                                    |


<a id="T_FAILURE">Failure object</a>

| KEY                | VALUE type                                                 | Description                                                                            |
|:-------------------|:-----------------------------------------------------------|:---------------------------------------------------------------------------------------|
| code               | [T_INT](#T_INT)                                            | [Failure code](#failure-code).                                                         |
| message            | [T_STRING](#T_STRING)                                      | Message for the failure.                                                               |

### icx_getTransactionByHash

Returns the transaction information requested by transaction hash.

> Request

```json
{
  "jsonrpc": "2.0",
  "id": "1001",
  "method": "icx_getTransactionByHash",
  "params": {
    "txHash": "0xd8da71e926052b960def61c64f325412772f8e986f888685bc87c0bc046c2d9f"
  }
}
```
#### Parameters

| KEY    | VALUE type        | Description             |
|:-------|:------------------|:------------------------|
| txHash | [T_HASH](#T_HASH) | Hash of the transaction |

> Example responses

```json
{
  "jsonrpc": "2.0",
  "result": {
    "blockHash": "0x8ef3b2a67262b9b1fe4b598059774472e9ccef401734335d87a4ba998cfd40fb",
    "blockHeight": "0x200",
    "from": "hx84f6c686fba03bc7ca65d15ae844ee56ff24a32b",
    "nid": "0x1",
    "signature": "tCUwOb6vsaUKy+NYvmzdJYC0jm3Erd5cR6wKnVuAjzMOECC+t/oK7fG/Tz2Y3C25o0AfCmbneXpias6xco+43wE=",
    "stepLimit": "0x3e8",
    "timestamp": "0x58a14bfe9b904",
    "to": "hx244deea00413d85c6637e7fdd53afa697f29d08f",
    "txHash": "0xd8da71e926052b960def61c64f325412772f8e986f888685bc87c0bc046c2d9f",
    "txIndex": "0x0",
    "value": "0xa",
    "version": "0x3"
  },
  "id": "1001"
}
```
#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

| KEY         | VALUE type                                                 | Description                                                                                             |
|:------------|:-----------------------------------------------------------|:--------------------------------------------------------------------------------------------------------|
| version     | [T_INT](#T_INT)                                            | Protocol version ("0x3" for V3)                                                                         |
| from        | [T_ADDR_EOA](#T_ADDR_EOA)                                  | EOA address that created the transaction                                                                |
| to          | [T_ADDR_EOA](#T_ADDR_EOA) or [T_ADDR_SCORE](#T_ADDR_SCORE) | EOA address to receive coins, or SCORE address to execute the transaction.                              |
| value       | [T_INT](#T_INT)                                            | Amount of ICX coins in loop to transfer. When omitted, assumes 0. (1 icx = 1 ^ 18 loop)                 |
| stepLimit   | [T_INT](#T_INT)                                            | Maximum step allowance that can be used by the transaction.                                             |
| timestamp   | [T_INT](#T_INT)                                            | Transaction creation time. Timestamp is in microsecond.                                                 |
| nid         | [T_INT](#T_INT)                                            | Network ID                                                                                              |
| nonce       | [T_INT](#T_INT)                                            | An arbitrary number used to prevent transaction hash collision.                                         |
| txHash      | [T_HASH](#T_HASH)                                          | Transaction hash                                                                                        |
| txIndex     | [T_INT](#T_INT)                                            | Transaction index in a block. Null when it is pending.                                                  |
| blockHeight | [T_INT](#T_INT)                                            | Block height where this transaction was in. Null when it is pending.                                    |
| blockHash   | [T_HASH](#T_HASH)                                          | Hash of the block where this transaction was in. Null when it is pending.                               |
| signature   | [T_SIG](#T_SIG)                                            | Signature of the transaction.                                                                           |
| dataType    | [T_DATA_TYPE](#T_DATA_TYPE)                                | Type of data. (call, deploy, message or deposit)                                                        |
| data        | JSON object                                                | Contains various type of data depending on the dataType. See [Parameters - data](#sendtxparameterdata). |

### icx_sendTransaction

You can do one of the followings using this function.
* Transfer designated amount of ICX coins from 'from' address to 'to' address.
* Install a new SCORE.
* Update the SCORE in the 'to' address.
* Invoke a function of the SCORE in the 'to' address.
* Transfer a message.
* Change deposit of the SCORE.

This function causes state transition.

> Coin transfer

```json
{
    "jsonrpc": "2.0",
    "method": "icx_sendTransaction",
    "id": 1234,
    "params": {
        "version": "0x3",
        "from": "hxbe258ceb872e08851f1f59694dac2558708ece11",
        "to": "hx5bfdb090f43a808005ffc27c25b213145e80b7cd",
        "value": "0xde0b6b3a7640000",
        "stepLimit": "0x12345",
        "timestamp": "0x563a6cf330136",
        "nid": "0x3",
        "nonce": "0x1",
        "signature": "VAia7YZ2Ji6igKWzjR2YsGa2m53nKPrfK7uXYW78QLE+ATehAVZPC40szvAiA6NEU5gCYB4c4qaQzqDh2ugcHgA="
    }
}
```

> SCORE function call

```json
{
    "jsonrpc": "2.0",
    "method": "icx_sendTransaction",
    "id": 1234,
    "params": {
        "version": "0x3",
        "from": "hxbe258ceb872e08851f1f59694dac2558708ece11",
        "to": "cxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32",
        "stepLimit": "0x12345",
        "timestamp": "0x563a6cf330136",
        "nid": "0x3",
        "nonce": "0x1",
        "signature": "VAia7YZ2Ji6igKWzjR2YsGa2m53nKPrfK7uXYW78QLE+ATehAVZPC40szvAiA6NEU5gCYB4c4qaQzqDh2ugcHgA=",
        "dataType": "call",
        "data": {
            "method": "transfer",
            "params": {
                "to": "hxab2d8215eab14bc6bdd8bfb2c8151257032ecd8b",
                "value": "0x1"
            }
        }
    }
}
```

> SCORE install

```json
{
    "jsonrpc": "2.0",
    "method": "icx_sendTransaction",
    "id": 1234,
    "params": {
        "version": "0x3",
        "from": "hxbe258ceb872e08851f1f59694dac2558708ece11",
        "to": "cx0000000000000000000000000000000000000000", // address 0 means SCORE install
        "stepLimit": "0x12345",
        "timestamp": "0x563a6cf330136",
        "nid": "0x3",
        "nonce": "0x1",
        "signature": "VAia7YZ2Ji6igKWzjR2YsGa2m53nKPrfK7uXYW78QLE+ATehAVZPC40szvAiA6NEU5gCYB4c4qaQzqDh2ugcHgA=",
        "dataType": "deploy",
        "data": {
            "contentType": "application/zip",
            "content": "0x1867291283973610982301923812873419826abcdef91827319263187263a7326e...", // compressed SCORE data
            "params": {  // parameters to be passed to on_install()
                "name": "ABCToken",
                "symbol": "abc",
                "decimals": "0x12"
            }
        }
    }
}
```

> SCORE update

```json
{
    "jsonrpc": "2.0",
    "method": "icx_sendTransaction",
    "id": 1234,
    "params": {
        "version": "0x3",
        "from": "hxbe258ceb872e08851f1f59694dac2558708ece11",
        "to": "cxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32", // SCORE address to be updated
        "stepLimit": "0x12345",
        "timestamp": "0x563a6cf330136",
        "nid": "0x3",
        "nonce": "0x1",
        "signature": "VAia7YZ2Ji6igKWzjR2YsGa2m53nKPrfK7uXYW78QLE+ATehAVZPC40szvAiA6NEU5gCYB4c4qaQzqDh2ugcHgA=",
        "dataType": "deploy",
        "data": {
            "contentType": "application/zip",
            "content": "0x1867291283973610982301923812873419826abcdef91827319263187263a7326e...", // compressed SCORE data
            "params": {  // parameters to be passed to on_update()
                "amount": "0x1234"
            }
        }
    }
}
```

> Message transfer

```json
{
    "jsonrpc": "2.0",
    "method": "icx_sendTransaction",
    "id": 1234,
    "params": {
        "version": "0x3",
        "from": "hxbe258ceb872e08851f1f59694dac2558708ece11",
        "to": "hxbe258ceb872e08851f1f59694dac2558708ece11",
        "stepLimit": "0x12345",
        "timestamp": "0x563a6cf330136",
        "nid": "0x3",
        "nonce": "0x1",
        "signature": "VAia7YZ2Ji6igKWzjR2YsGa2m53nKPrfK7uXYW78QLE+ATehAVZPC40szvAiA6NEU5gCYB4c4qaQzqDh2ugcHgA=",
        "dataType": "message",
        "data": "0x4c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e7365637465747572206164697069736963696e6720656c69742c2073656420646f20656975736d6f642074656d706f7220696e6369646964756e74207574206c61626f726520657420646f6c6f7265206d61676e6120616c697175612e20557420656e696d206164206d696e696d2076656e69616d2c2071756973206e6f737472756420657865726369746174696f6e20756c6c616d636f206c61626f726973206e69736920757420616c697175697020657820656120636f6d6d6f646f20636f6e7365717561742e2044756973206175746520697275726520646f6c6f7220696e20726570726568656e646572697420696e20766f6c7570746174652076656c697420657373652063696c6c756d20646f6c6f726520657520667567696174206e756c6c612070617269617475722e204578636570746575722073696e74206f6363616563617420637570696461746174206e6f6e2070726f6964656e742c2073756e7420696e2063756c706120717569206f666669636961206465736572756e74206d6f6c6c697420616e696d20696420657374206c61626f72756d2e"
    }
}
```

> Deposit add
```json
{
    "jsonrpc": "2.0",
    "method": "icx_sendTransaction",
    "id": 1234,
    "params": {
        "version": "0x3",
        "from": "hxbe258ceb872e08851f1f59694dac2558708ece11",
        "timestamp": "0x563a6cf330136",
        "to": "cx2f501ff91ad48732673adf55a04f36d466cf269c",
        "stepLimit": "0x50000000",
        "nid": "0x3",
        "nonce": "0x1",
        "value": "0x10f0cf064dd59200000",
        "dataType": "deposit",
        "data": {
            "action": "add",
        }
    }
}
```

#### Parameters

| KEY       | VALUE type                                                 | Required | Description                                                                                          |
|:----------|:-----------------------------------------------------------|:--------:|:-----------------------------------------------------------------------------------------------------|
| version   | [T_INT](#T_INT)                                            | required | Protocol version ("0x3" for V3)                                                                      |
| from      | [T_ADDR_EOA](#T_ADDR_EOA)                                  | required | EOA address that created the transaction                                                             |
| to        | [T_ADDR_EOA](#T_ADDR_EOA) or [T_ADDR_SCORE](#T_ADDR_SCORE) | required | EOA address to receive coins, or SCORE address to execute the transaction.                           |
| value     | [T_INT](#T_INT)                                            | optional | Amount of ICX coins in loop to transfer. When omitted, assumes 0. (1 icx = 1 ^ 18 loop)              |
| stepLimit | [T_INT](#T_INT)                                            | required | Maximum step allowance that can be used by the transaction.                                          |
| timestamp | [T_INT](#T_INT)                                            | required | Transaction creation time. Timestamp is in microsecond.                                              |
| nid       | [T_INT](#T_INT)                                            | required | Network ID ("0x1" for Mainnet, "0x2" for Testnet, etc)                                               |
| nonce     | [T_INT](#T_INT)                                            | optional | An arbitrary number used to prevent transaction hash collision.                                      |
| signature | [T_SIG](#T_SIG)                                            | required | Signature of the transaction.                                                                        |
| dataType  | [T_DATA_TYPE](#T_DATA_TYPE)                                | optional | Type of data. (call, deploy, message or deposit)                                                     |
| data      | JSON object                                                | optional | The content of data varies depending on the dataType. See [Parameters - data](#sendtxparameterdata). |

#### <a id ="sendtxparameterdata">Parameters - data</a>
`data` contains the following data in various formats depending on the dataType.

##### dataType == call

It is used when calling a function in SCORE, and `data` has dictionary value as follows.

| KEY    | VALUE type  | Required | Description                             |
|:-------|:------------|:--------:|:----------------------------------------|
| method | String      | required | Name of the function to invoke in SCORE |
| params | JSON object | optional | Function parameters                     |

##### dataType == deploy

It is used when installing or updating a SCORE, and `data` has dictionary value as follows.

| KEY         | VALUE type                | Required | Description                                                          |
|:------------|:--------------------------|:--------:|:---------------------------------------------------------------------|
| contentType | String                    | required | Mime-type of the content                                             |
| content     | [T_BIN_DATA](#T_BIN_DATA) | required | Compressed SCORE data                                                |
| params      | JSON object               | optional | Function parameters will be delivered to on_install() or on_update() |

##### dataType == message

It is used when transferring a message, and `data` has a HEX string.

##### dataType == deposit

It is used to change deposit.

| KEY    | VALUE type        | Required | Description                     |
|:-------|:------------------|:--------:|:--------------------------------|
| action | String            | required | Action to do. ( add, withdraw ) |
| id     | [T_HASH](#T_HASH) | optional | ID of the deposit to withdraw   |
| amount | [T_INT](#T_INT)   | optional | Amount of deposit to withdraw   |

While the `action` is `add`, it uses coin value for adding a limited deposit or
increasing the unlimited deposit.
The deposit would be used for paying used steps by the transaction.

While the `action` is `withdraw`, `id` must be set for the limited deposit.
Otherwise, it may set `amount` for partial withdrawal for the unlimited deposit.

| Case                                 | data.action | data.id           | data.amount        | value         |
|:-------------------------------------|:------------|:-----------------:|:-------------------|:--------------|
| Add deposit                          | `add`       |                   |                    | amount to add |
| Withdraw limited deposit             | `withdraw`  | ID of the deposit |                    |               |
| Withdraw a part of unlimited deposit | `withdraw`  |                   | amount to withdraw |               |
| Withdraw whole of unlimited deposit  | `withdraw`  |                   |                    |               |


> Example responses

```json
{
  "jsonrpc": "2.0",
  "result": "0x402b630c5ed80d1b8f0d89ca14a091084bcc0f6a98bc52329bccc045415bc0bd",
  "id": "1001"
}
```

#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

* Transaction hash ([T_HASH](#T_HASH)) on success
* Error code and message on failure


### icx_sendTransactionAndWait

It sends a transaction like `icx_sendTransaction`, then it will wait for the
result of it for specified time. If the timeout isn't set by user, it uses
`defaultWaitTimeout`.

It's disabled by default. It can be enabled by setting `defaultWaitTimeout` as none-zero value.

#### Responses


| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

* Same response value([Transaction Result](#T_RESULT)) as `icx_getTransactionResult` on success
* Error code, message and data on failure
* `data` field of failure will be transaction hash([T_HASH](#T_HASH)) on timeout


### icx_waitTransactionResult

It will wait for the result of the transaction for specified time.
If the timeout isn't set by user, it uses `defaultWaitTimeout`.

It's disabled by default. It can be enabled by setting `defaultWaitTimeout` as none-zero value.

#### Responses


| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

* Same response value([Transaction Result](#T_RESULT)) as `icx_getTransactionResult` on success
* Error code, message and data on failure
* `data` field of failure will be transaction hash([T_HASH](#T_HASH)) on timeout

### icx_getScoreStatus

It returns status information of the smart contract.

> Request
```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getScoreStatus",
  "params": {
    "address": "cxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32"
  }
}
```
#### Parameters

| KEY     | VALUE type                    | Required | Description                   |
|:--------|:------------------------------|:---------|:------------------------------|
| address | [T_ADDR_SCORE](#T_ADDR_SCORE) | required | SCORE address to be examined. |
| height  | [T_INT](#T_INT)               | optional | Integer of a block height     |

> Example responses
```json
{
  "jsonrpc": "2.0",
  "id": 1001,
  "result": {
      "current": {
        "auditTxHash": "0x5ba8712782563fec86bbd6381a5a38c40ed74fc945f2f5c43321354d66343c0a",
        "codeHash": "0x7c7e4e67727a5f6c11f03dab37333e50ed6d47c243b4e486eaaa05d407fd3c84",
        "deployTxHash": "0x5ba8712782563fec86bbd6381a5a38c40ed74fc945f2f5c43321354d66343c0a",
        "type": "python",
        "status": "active"
      },
      "depositInfo": {
        "availableDeposit": "0x10f0cf064dd59200000",
        "availableVirtualStep": "0x0",
        "deposits": [
          {
            "depositRemain": "0x10f0cf064dd59200000"
          }
        ]
      },
      "owner": "hxff9221db215ce1a511cbe0a12ff9eb70be4e5764",
      "useSystemDeposit": "0x1"
  }
}
```
#### Response

| Status | Meaning | Description | Schema      |
|:-------|:--------|:------------|:------------|
| 200    | OK      | Success     | ScoreStatus |

* [SCORE Status](#T_SCORE_STATUS) as result on success
* Error code, message and data on failure
* Given address isn't valid contract address, it returns failure.

<a id="T_SCORE_STATUS">SCORE Status</a>

| KEY              | VALUE type                          | Description                         |
|:-----------------|:------------------------------------|:------------------------------------|
| owner            | [T_ADDR_SCORE](#T_ADDR_SCORE)       | Owner of the score                  |
| blocked          | [T_BOOL](#T_BOOL)                   | `0x1` if it's blocked by governance |
| disabled         | [T_BOOL](#T_BOOL)                   | `0x1` if it's disabled by owner     |
| useSystemDeposit | [T_BOOL](#T_BOOL)                   | `0x1` if it uses system deposit     |
| current          | [Contract Status](#ContractStatus)  | Current contract                    |
| next             | [Contract Status](#ContractStatus)  | Next contract to be audited         |
| depositInfo      | [Deposit Information](#DepositInfo) | Deposit information                 |


<a id="ContractStatus">Contract Status</a>

| KEY          | VALUE type            | Description                                  |
|:-------------|:----------------------|:---------------------------------------------|
| status       | [T_STRING](#T_STRING) | Status of the contract                       |
| deployTxHash | [T_HASH](#T_HASH)     | TX Hash for deploy                           |
| auditTxHash  | [T_HASH](#T_HASH)     | TX Hash for audit                            |
| type         | [T_STRING](#T_STRING) | Type of the code (one of system,java,python) |
| codeHash     | [T_HASH](#T_HASH)     | Hash of the code                             |


<a id="DepositInfo">Deposit Information</a>

| KEY                  | VALUE type                     | Description                         |
|:---------------------|:-------------------------------|:------------------------------------|
| availableDeposit     | [T_INT](#T_INT)                | Available deposit amount            |
| availableVirtualStep | [T_INT](#T_INT)                | Available virtual steps(deprecated) |
| deposits             | a list of [Deposit](#Deposit)s | Remaining deposits                  |


<a id="Deposit">Deposit</a>

* Deposit V1

| KEY               | VALUE type        | Description              |
|:------------------|:------------------|:-------------------------|
| id                | [T_HASH](#T_HASH) | ID of deposit            |
| depositRemain     | [T_INT](#T_INT)   | Available deposit amount |
| depositUsed       | [T_INT](#T_INT)   | Used deposit amount      |
| expires           | [T_INT](#T_INT)   | Expiration block height  |
| virtualStepIssued | [T_INT](#T_INT)   | Issued virtual steps     |
| virtualStepUsed   | [T_INT](#T_INT)   | Used virtual steps       |


* Deposit V2

| KEY           | VALUE type      | Description              |
|:--------------|:----------------|:-------------------------|
| depositRemain | [T_INT](#T_INT) | Available deposit amount |


### icx_getNetworkInfo

It returns basic network information

>Request
```json
{
  "id": 1002,
  "jsonrpc": "2.0",
  "method": "icx_getNetworkInfo"
}
```

#### Response

| Status | Meaning | Description | Schema                                 |
|:-------|:--------|:------------|:---------------------------------------|
| 200    | OK      | Success     | [Network Information](#T_NETWORK_INFO) |

* [Network Information](#T_NETWORK_INFO) as result on success
* Error code, message and data on failure


<a id="T_NETWORK_INFO">Network Information</a>

| KEY       | VALUE type            | Description                          |
|:----------|:----------------------|:-------------------------------------|
| platform  | [T_STRING](#T_STRING) | Name of the platform                 |
| nid       | [T_INT](#T_INT)       | Network ID of the current channel    |
| channel   | [T_STRING](#T_STRING) | Name of the current channel          |
| earliest  | [T_INT](#T_INT)       | Height of the earliest block         |
| latest    | [T_INT](#T_INT)       | Height of the latest finalized block |
| stepPrice | [T_INT](#T_INT)       | Price of the step                    |


## JSON-RPC Debug

The debug end point is `http://<host>:<port>/api/v3d/<channel>`

A rule for channel name in main end point is applied.

APIs for debug endpoint.
* [debug_estimateStep](#debug_estimatestep)
* [debug_getTrace](#debug_gettrace)

### debug_getTrace

Returns the trace logs of the transaction

> Request

```json
{
  "jsonrpc": "2.0",
  "id": "1001",
  "method": "debug_getTrace",
  "params": {
    "txHash": "0x4f4feed4a1d29779f84460d663e1ffb894d65dacfa3cc215a353a4b0d0d8f020"
  }
}
```

#### Parameters

| KEY  | VALUE type        | Required | Description                   |
|:-----|:------------------|:---------|:------------------------------|
| hash | [T_HASH](#T_HASH) | required | Hash value of the transaction |

> Example responses

```json
{
  "jsonrpc": "2.0",
  "result": {
    "logs": [
      {
        "level": 2,
        "msg": "FRAME[1] TRANSACTION start from=hx92b7608c53825241069a280982c4d92e1b228c84 to=cx9e3cadcc1a4be3323ea23371b84575abb32703ae id=0x4f4feed4a1d29779f84460d663e1ffb894d65dacfa3cc215a353a4b0d0d8f020",
        "ts": 0
      },
      {
        "level": 2,
        "msg": "FRAME[1] STEP apply type=default count=1 cost=100000 total=100000",
        "ts": 108
      },
      {
        "level": 2,
        "msg": "FRAME[1] STEP apply type=input count=101 cost=20200 total=120200",
        "ts": 112
      },
      {
        "level": 2,
        "msg": "FRAME[1] TRANSACTION charge fee=2054062500000000 steps=164325 price=12500000000",
        "ts": 1097
      },
      {
        "level": 2,
        "msg": "FRAME[1] TRANSACTION done status=Success steps=164325 price=12500000000",
        "ts": 1108
      }
    ],
    "status": "0x1"
  },
  "id": 100
}
```

#### Responses

| Status | Meaning | Description | Schema |
|:-------|:--------|:------------|:-------|
| 200    | OK      | Success     | Block  |

<a id="T_TRACELOGS">Trace Logs</a>

| KEY  | VALUE type | Description                       |
|:-----|:-----------|:----------------------------------|
| logs | JSON array | Array of [Trace Log](#T_TRACELOG) |

<a id="T_TRACELOG">Trace Log</a>

| KEY   | VALUE type  | Description                                    |
|:------|:------------|:-----------------------------------------------|
| level | JSON number | Log level(0:Trace, 1:Debug, 2:System)          |
| msg   | JSON string | Log message                                    |
| ts    | JSON number | Time offset from the beginning in micro-second |

### debug_estimateStep

* Returns an estimated step of how much step is necessary to allow the transaction to complete. The transaction will not be added to the blockchain. Note that the estimation can be larger than the actual amount of step to be used by the transaction for several reasons such as node performance.

> Request
```json
{
  "jsonrpc": "2.0",
  "method": "debug_estimateStep",
  "id": 1234,
  "params": {
    "version": "0x3",
    "from": "hxbe258ceb872e08851f1f59694dac2558708ece11",
    "to": "hx5bfdb090f43a808005ffc27c25b213145e80b7cd",
    "value": "0xde0b6b3a7640000",
    "timestamp": "0x563a6cf330136",
    "nid": "0x3",
    "nonce": "0x1"
  }
}
```

#### Parameters

* The transaction information without stepLimit and signature

| KEY       | VALUE type                                                 | Required | Description                                                                                          |
|:----------|:-----------------------------------------------------------|:--------:|:-----------------------------------------------------------------------------------------------------|
| version   | [T_INT](#T_INT)                                            | required | Protocol version ("0x3" for V3)                                                                      |
| from      | [T_ADDR_EOA](#T_ADDR_EOA)                                  | required | EOA address that created the transaction                                                             |
| to        | [T_ADDR_EOA](#T_ADDR_EOA) or [T_ADDR_SCORE](#T_ADDR_SCORE) | required | EOA address to receive coins, or SCORE address to execute the transaction.                           |
| value     | [T_INT](#T_INT)                                            | optional | Amount of ICX coins in loop to transfer. When ommitted, assumes 0. (1 icx = 1 ^ 18 loop)             |
| timestamp | [T_INT](#T_INT)                                            | required | Transaction creation time. timestamp is in microsecond.                                              |
| nid       | [T_INT](#T_INT)                                            | required | Network ID ("0x1" for Mainnet, "0x2" for Testnet, etc)                                               |
| nonce     | [T_INT](#T_INT)                                            | optional | An arbitrary number used to prevent transaction hash collision.                                      |
| dataType  | [T_DATA_TYPE](#T_DATA_TYPE)                                | optional | Type of data. (call, deploy, or message)                                                             |
| data      | JSON dict or JSON string                                   | optional | The content of data varies depending on the dataType. See [Parameters - data](#sendtxparameterdata). |

#### Response

* The amount of an estimated step

> Response - success
```json
{
    "jsonrpc": "2.0",
    "id": 1234,
    "result": "0x109eb0"
}

```

> Response - failure
```json
{
    "jsonrpc": "2.0",
    "id": 1234,
    "error": {
        "code": -32602,
        "message": "JSON schema validation error: 'version' is a required property"
    }
}
```
