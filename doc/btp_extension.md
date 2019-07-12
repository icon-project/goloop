---
title: Extension for BTP
---
# Extension for BTP

## Introduction

Blockchain system requirements of BTP are described in the document

Summarize the document to following items.

* A method to get the receipt with the proof path from the block
    * The block contains the root of receipts
    * API to get receipt including proof
* A method to detect validator change.
    * The block contains the root of next validators.
    * API to get the block
    * API to monitor the blockchain
* A method to get validators related with the block
    * The block contains the root of next validators
    * API to get validators for the root
* A method to get votes for the block
    * API to get votes for the block
* A method to detect events
    * The block contains logsbloom related to events.
    * API to monitor events

## Monitor with Websocket

### Block

`GET /api/v3/:channel/block`

> Request

```json
{
  "height": "0x10"
}
```
#### Parameters

|Name|Type|Required|Description|
|---|---|---|---|
|height|T_INT|true| Start height |

> Success Responses

```json
{
  "code": 0
}
```

> Failure Response

```json
{
  "code": -32602,
  "message": "Bad params"
}
```

#### Responses

|Name|Type|Required|Description|
|---|---|---|---|
|code|Number|true| 0 or JSON RPC error code. 0 means success. |
|message|String|false| error message. |

> Example notification

```json
{
  "hash": "0xabc...",
  "height": "0x10"
}
```

#### Notification

|Name|Type|Required|Description|
|---|---|---|---|
|hash|T_HASH|true|The hash of the new block|
|height|T_INT|true|The height of the new block|

### Events

`GET /api/v3/:channel/event`

> Request

```json
{
  "height": "0x10",
  "addr": "cx49894fa5aec4d662e49934f297673cf08dd9f382",
  "event": "Event(int,bytes,int,Address)",
  "data": [
      null,
      "0xda12",
      "0x2",
      "hxb51a65420ce5199e538f21fc614eacf4234454fe"
  ]
}
```
#### Parameters

|Name|Type|Required|Description|
|---|---|---|---|
|height|T_INT|true| Start height |
|addr|T_ADDR|false||
|event|String|true| Event signature |
|data|Array|false| Array of parameters to match. Its length shall be equal to the number of parameters of the event. `null` matches any value. |

> Success Responses

```json
{
  "code": 0
}
```

> Failure Response

```json
{
  "code": -32602,
  "message": "Bad params"
}
```

#### Responses

|Name|Type|Required|Description|
|---|---|---|---|
|code|Number|true| 0 or JSON RPC error code. 0 means success. |
|message|String|false| error message. |


> Example notifiaction

```json
{
  "hash": "0xdbc...",
  "height": "0x11",
  "index":  "0"
}
```

#### Notification

|Name|Type|Required|Description|
|---|---|---|---|
|hash|T_HASH|true|Hash of the block including the events|
|height|T_INT|true|Height of the block including the events|
|index|T_INT|true|Index of the result including the events in the block|


You may use `hash` and `index` to get proof of the result including
the events(`icx_getProofForResult`).


## Extended JSON-RPC Methods

### icx_getDataByHash

Get data by hash.

It can be used to retrieve data based on the hash algorithm (SHA3-256).

Following data can be retrieved by a hash.

* BlockHeader with the hash of the block
* Validators with BlockHeader.NextValidatorHash
* Votes with BlockHeader.VotesHash
* etc…

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getDataByHash",
  "params": {
      "hash": "0x1fcf7c34dc875681761bdaa5d75d770e78e8166b5c4f06c226c53300cbe85f57"
  }
}
```
#### Parameters

|Name|Type|Required|Description|
|---|---|---|---|
|hash|T_HASH|true|The hash value of the data to retrieve.|


> Example responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": ""
}
```

> default Response

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Something went wrong."
  }
}
```

#### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|OK|Success|Data : base64 encoded bytes|
|default|Default|JSON-RPC Error|Error Response|

### icx_getBlockHeaderByHeight

Get block header for specified height.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getBlockHeaderByHeight",
  "params": {
      "height": "0x10"
  }
}
```
#### Parameters

|Name|Type|Required|Description|
|---|---|---|---|
|height|T_INT|true|The height of the block in hex string.|

> Example responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": ""
}
```

> default Response

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Something went wrong."
  }
}
```

#### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|OK|Success|Data : base64 encoded bytes|
|default|Default|JSON-RPC Error|Error Response|

### icx_getVotesByHeight

Get votes for the block specified by height.

Normally votes for the block are included in the next. So, even though the block is finalized by votes already, the block including votes may not exist. For that reason, we support this API to get votes as proof for the block.


> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getVotesByHeight",
  "params": {
      "height": "0x10"
  }
}
```
#### Parameters

|Name|Type|Required|Description|
|---|---|---|---|
|height|T_INT|true|The height of the block for votes.|

> Example responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": ""
}
```

> default Response

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Something went wrong."
  }
}
```

#### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|OK|Success|Encoded votes|
|default|Default|JSON-RPC Error|Error Response|

### icx_getProofForResult

Get proof for the receipt. Proof, itself, may include the receipt.

Currently, Core2 uses Merkle Patricia Trie to store receipt, so the last leaf node includes the receipt. Key for the receipt must be the binary representation of the unsigned integer, the index of the receipt.


> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getProofForResult",
  "params": {
      "hash": "0xc7fae616bd1d377a92c48a35e33e7a072e5e2be155c000088dbdd42a3e31bb74",
      "index": "0x0"
  }
}
```
#### Parameters

|Name|Type|Required|Description|
|---|---|---|---|
|hash|T_HASH|true|The hash value of the block including the result.|
|index|T_INT|true|Index of the receipt in the block.<br/> 0 for the first.|

> Example responses
```json
{
   "id": 1001,
   "jsonrpc": "2.0",
   "result": [
     "+QExoJezi7p+Q6Sj6RP1PYooLRlGqEz/UWwr0/VryoVpsmVAoLNM357mKnozSX9XP5PASGC4bq+QLpEY3Sg8xdIE7zD3oKIIH/jyDGldEmFNolkwa+3cxEJdrXWItDra84U66s9NoIfRLI8juWLG+6Z/xGVX1g3+J4yDEeInX0gABWGti5euoJrK+4GdphdGbxY4WsLIrflkcfrcqYGmazXClEtYjQ37oIiIPWzA4QFh/R/W1cuHM46RipVLzsSOdLPaFPOeBajToCXRQQ4GVjQiKXMgCTPPaqs30SMfDTTQpMiu9EWhB7wzoKxysxQEI6koHy7gnF5ob9nZYP7QDF/zV7MHU7Ut6NuRgICAgKDdQ/iagBv+CtzgCh1ju+SjVjZgOVeu6BZfBaAs5hYDo4CAgIA=",
     "+QIRoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoCIc9dReCXYR967Ll8MBSUxzksWDY2BnoQi9Wd/7oEoWoPkCx+uBkmGXMdfppwKUS/jaqLBEcxWj4bVoq/WpxFRzoJBir1eJCOvvqV9urYfxHvZ9E4MTcrb9Or7uLXyOQN78oB9ED5ht8egUlm/SGXX1UlpRFz+VwwgN6EY2TH8LJUT7oKsA5iI9WcteAH3ApzQCwO9BGpSHECr7Od0DEGf9/IxAoOsZFmn1IS2/EGAB97IbYRQGIy3j19DS2Y0jWyNmyT5XoERkVHKeInAzSMZcSm22AIIawXF/ibDdskyEDabbdnO5oCxrQAjl/71HrhhG7jokBsviGC3RYglC34NbtOWzZaoHoJMWXQn5I+cRmWg76pmT8VrDO0DSWGMyv1X3GbkPo8w/oPEBG9Q+RjtCMovVi9K6XG08khJpsPtcHB6YkOlHTLa8oPPEZm2q+9Cssdo5l0YzKH7/+cV1h5pxp8baWeUUUssFoBIHc9BwAGJDsArHrh9kkvS6K8B6xmOzRDR0eKfzC9NcoFHqm63YUFSq9I+9gVJB+VDPGWvp6ZV1AejoXwXS/8rkoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoJl4/9qlwu2vrYvpyQ8ayLvfMOd3Tmc3KZT7FTTfJjJ3gA==",
     "6CCmmADEFQCrJP7Wvhjk9aBRoZai37jZw23jIMQBAMQBAMQBAMQAwMA="
   ]
}
```

> Failure Response
```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Something went wrong."
  }
}
```

#### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|OK|Success|List of base64 encoded proof including the receipt|
|default|Default|JSON-RPC Error|Error Response|


## Binary format

Core2 uses MsgPack and RLP for binary encoding and decoding.

* [MsgPack](https://msgpack.org)
* [RLP](https://github.com/ethereum/wiki/wiki/RLP)

### Block Header

|Name|Field|Type|Description|
|---|---|---|---|
|BlockHeader||MsgPack List of followings||
||Version|MsgPack Int|1 ← Version 1 (legacy)<br/>2 ← Version 2 (core2 beta)|
||Height|MsgPack Int|Height of the block.<br/>0 means genesis block.|
||Timestamp|MsgPack Int|Micro-seconds after EPOCH.|
||Proposer|Address|Height of the block.<br/>0 means genesis block.|
||PrevID|MsgPack Bytes|32 bytes hash value|
||VotesHash|MsgPack Bytes|32 bytes hash value|
||NextValidatorHash|MsgPack Bytes|32 bytes hash value|
||PatchTransactionHash|MsgPack Bytes|32 bytes hash value|
||NormalTransactionHash|MsgPack Bytes|32 bytes hash value|
||LogsBloom|MsgPack Bytes|N(1~256) bytes bloom log value|
||Result|MsgPack Bytes|Result.Encode()<br/>After decoding BlockHeader, it should decode it again for NormalReceiptHash.|
|Result     ||MsgPack List of followings||
||StateHash|MsgPack Bytes|Hash of world state (account information)|
||PatchReceiptHash|MsgPack Bytes|Root Hash of patch receipts|
||NormalReceiptHash|MsgPack Bytes|Root Hash of normal receipts|

### Validators

|Name|Field|Type|Description|
|---|---|---|---|
|Validators||MsgPack List of Vadidator||
|Validator||MsgPack Bytes|21 bytes → same as Address<br/>Other bytes → public key|

### Votes

|Name|Field|Type|Description|
|---|---|---|---|
|Votes||MsgPack List of followings||
||Round|MsgPack Int|Round for votes.<br/>If consensus doesn’t use round, it should be 0(zero).|
||BlockPartSetID|PartSetID|If it doesn’t use PartSetID, it should be empty list.|
||Items|MsgPack List of VoteItem||
|VoteItem||MsgPack List of followings||
||Timestamp|MsgPack Int||
||Signature|Signature||
|PartSetID||MsgPack List of followings||
||Count|MsgPack Unsigned Int|Number of block parts|
||Hash|MsgPack Bytes|Hash of block parts|
|Signature||MsgPack Bytes|RSV format signature for VoteMessage|
|VoteMessage||MsgPack List of followings||
||Height|MsgPack Int|BlockHeader.Height|
||Round|MsgPack Int|Votes.Round|
||Type|MsgPack Int|0 ← PreVote ( only for consensus )<br/>1 ← PreCommit ( for vote check )|
||BlockID|MsgPack Bytes|SHA3Sum256(BlockHeader)|
||BlockPartSetID|PartSetID|Votes.BlockPartSetID.|
||Timestamp|MsgPack Int||

### Proof

|Name|Field|Type|Description|
|---|---|---|---|
|Proof||MPT Node||
|MPT Node||MPT Leaf<br/>MPT Extension<br/>MPT Branch|If the number of elements is 17, then it’s MPT Branch.<br/>It differentiates MPT Leaf from MPT Extension with a prefix in a header.|
|MPT Leaf||RLP List of followings||
||Header|RLP Bytes|N bytes ( Prefix + Nibbles )|
||Value|RLP Bytes|N bytes ( Receipt )|
|MPT Extension||RLP List of followings||
||Header|RLP Bytes|N bytes ( Prefix + Nibbles )|
||Link|RLP Bytes<br/>MPT Node|If encoded MPT Node is shorter than 32, then it’s embedded.<br/>Otherwise, it uses RLP Bytes for sha3sum256 value|
|MPT Branch||RLP List of followings||
||Link x 16|RLP Bytes<br/>MPT Node|If encoded MPT Node is shorter than 32, then it’s embedded.<br/>Otherwise, it uses RLP Bytes for sha3sum256 value|
||Value|RLP Bytes|N bytes ( Data )|

### Receipt

|Name|Field|Type|Description|
|---|---|---|---|
|Receipt||MsgPack List of followings||
||Status|MsgPack Int|Result status<br/>0 ← SUCCESS<br/>N ← FAILURE ( N is failure code )|
||To|Address|The target address of the transaction|
||CumulativeStepUsed|Integer|Cumulative step used|
||StepUsed|Integer|Step used|
||StepPrice|Integer|Step price in LOOP|
||LogsBloom|Integer|2048 bits without padding zeros<br/>So, if there is no bit, then it would be a byte with zero.|
||EventLogs|MsgPack List of EventLog||
||SCOREAddress|Address||
|EventLog||MsgPack List of followings||
||Addr|Address|SCORE producing this event log|
||Indexed|MsgPack List of MsgPack Bytes|Indexed data.|
||Data|MsgPack List of MsgPack Bytes|Remaining data.|
|Address||MsgPack Bytes|1 byte<br/>- 0x00 ← EOA<br/>- 0x01 ← SCORE<br/>20 bytes : Identifier|
|Integer||MsgPack Bytes|N bytes of integer representation.<br/>ex)<br/>0x00 → [ 0x00 ]<br/>0x80 → [ 0x00 0x80 ]<br/>-0x01 → [ 0xff ]<br/>-0x80 → [ 0x80 ]|

