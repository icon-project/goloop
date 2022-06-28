---
title: Extension for BTP 2.0
---
# Extension for BTP 2.0

## Introduction
This document specifies the design of Server RPC API for BTP 2.0.

Summarize the document to following items.


## Monitor with Websocket

### Block

`GET /api/v3/:channel/btp`

> Request

```json
{
  "height": "0x10",
  "networkID": "0x1",
  "proofFlag": "0x1"
}
```
#### Parameters

| Name      | Type     | Required    | Description         |
|:----------|:---------|:------------|:--------------------|
| height    | T_INT    | true        | Start height        |
| networkID | T_INT    | true        | Network ID          |
| proofFlag | T_INT    | true        | Proof included flag |
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

| Name    | Type   | Required | Description                                |
|:--------|:-------|:---------|:-------------------------------------------|
| code    | Number | true     | 0 or JSON RPC error code. 0 means success. |
| message | String | false    | Error message.                             |


> Example notification

```json
{
  "btpHeader" : "+QIRoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoCIc9dReCXYR967Ll8MBSUxzksWDY2BnoQi9Wd/7oEoWoPkCx+uBkmGXMdfppwKUS/jaqLBEcxWj4bVoq/WpxFRzoJBir1eJCOvvqV9urYfxHvZ9E4MTcrb9Or7uLXyOQN78oB9ED5ht8egUlm/SGXX1UlpRFz+VwwgN6EY2TH8LJUT7oKsA5iI9WcteAH3ApzQCwO9BGpSHECr7Od0DEGf9/IxAoOsZFmn1IS2/EGAB97IbYRQGIy3j19DS2Y0jWyNmyT5XoERkVHKeInAzSMZcSm22AIIawXF/ibDdskyEDabbdnO5oCxrQAjl/71HrhhG7jokBsviGC3RYglC34NbtOWzZaoHoJMWXQn5I+cRmWg76pmT8VrDO0DSWGMyv1X3GbkPo8w/oPEBG9Q+RjtCMovVi9K6XG08khJpsPtcHB6YkOlHTLa8oPPEZm2q+9Cssdo5l0YzKH7/+cV1h5pxp8baWeUUUssFoBIHc9BwAGJDsArHrh9kkvS6K8B6xmOzRDR0eKfzC9NcoFHqm63YUFSq9I+9gVJB+VDPGWvp6ZV1AejoXwXS/8rkoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoJl4/9qlwu2vrYvpyQ8ayLvfMOd3Tmc3KZT7FTTfJjJ3gA=="
}
```

#### Notification

| Name   | Type      | Description                                      |
|:-------|:----------|:-------------------------------------------------|
| header | T_BASE64  | Base64 encoded [BTPBlockHeader](#btpblockheader) |
| proof  | T_BASE64  | Base64 encoded proof                             |


## BTP JSON-RPC Methods

### btp_getNetworkInfo

Get BTP network information.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "btp_getNetworkInfo",
  "params": {
    "id" : "0x3"
  }
}
```
#### Parameters

| Name   | Type    | Required | Description       |
|:-------|:--------|:---------|:------------------|
| height | T_INT   | false    | Main block height |
| id     | T_INT   | true     | Network ID        |


> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": {
    "startHeight" : "0x11",
    "networkTypeID" : "0x1",
    "networkTypeName" : "eth",
    "networkID" : "0x3",
    "networkName" : "snow",
    "open": "0x1",
    "nextMessageSN" : "0x20",
    "prevNSHash" : "0x…",
    "lastNSHash" : "0x…"
  }
}
```
#### Responses

| Name            | Type      | Description                          |
|:----------------|:----------|:-------------------------------------|
| startHeight     | T_INT     | Block height where BTP block started |
| networkTypeID   | T_INT     | Network type ID                      |
| networkTypeName | T_STRING  | Network type name                    |
| networkID       | T_INT     | Network ID                           |
| networkName     | T_STRING  | Network name                         |
| open            | T_INT     | Active state of network              |
| nextMessageSN   | T_INT     | Next message SN                      |
| prevNSHash      | T_HASH    | Previous network hash                |
| lastNSHash      | T_HASH    | Last network hash                    |

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

#### Default Responses

| Status  | Meaning | Description    | Schema                      |
|:--------|:--------|:---------------|:----------------------------|
| 200     | OK      | Success        | Data : base64 encoded bytes |
| default | Default | JSON-RPC Error | Error Response              |


### btp_getNetworkTypeInfo

Get BTP network type information.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getNetworkTypeInfo",
  "params": {
    "id" : "0x02"
  }
}
```
#### Parameters

| Name   | Type    | Required | Description       |
|:-------|:--------|:---------|:------------------|
| height | T_INT   | false    | Main block height |
| id     | T_INT   | true     | Network type ID   |



> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": {
    "networkTypeID" : "0x2",
    "networkTypeName" : "eth",
    "openNetworkIDs" : ["0x3","0x4"],
    "nextProofContext" : "+QIRoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoCIc9dReCXYR967Ll8MBSUxzksWDY2BnoQi9Wd/7oEoWoPkCx+uBkmGXMdfppwKUS/jaqLBEcxWj4bVoq/WpxFRzoJBir1eJCOvvqV9urYfxHvZ9E4MTcrb9Or7uLXyOQN78oB9ED5ht8egUlm/SGXX1UlpRFz+VwwgN6EY2TH8LJUT7oKsA5iI9WcteAH3ApzQCwO9BGpSHECr7Od0DEGf9/IxAoOsZFmn1IS2/EGAB97IbYRQGIy3j19DS2Y0jWyNmyT5XoERkVHKeInAzSMZcSm22AIIawXF/ibDdskyEDabbdnO5oCxrQAjl/71HrhhG7jokBsviGC3RYglC34NbtOWzZaoHoJMWXQn5I+cRmWg76pmT8VrDO0DSWGMyv1X3GbkPo8w/oPEBG9Q+RjtCMovVi9K6XG08khJpsPtcHB6YkOlHTLa8oPPEZm2q+9Cssdo5l0YzKH7/+cV1h5pxp8baWeUUUssFoBIHc9BwAGJDsArHrh9kkvS6K8B6xmOzRDR0eKfzC9NcoFHqm63YUFSq9I+9gVJB+VDPGWvp6ZV1AejoXwXS/8rkoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoJl4/9qlwu2vrYvpyQ8ayLvfMOd3Tmc3KZT7FTTfJjJ3gA=="
  }
}
```
#### Responses

| Name             | Type             | Description                         |
|:-----------------|:-----------------|:------------------------------------|
| networkTypeID    | T_INT            | Network type ID                     |
| networkTypeName  | T_STRING         | Network type name                   |
| openNetworkIDs   | T_ARRAY of T_INT | Network ID included in network type |
| nextProofContext | T_BASE64         | Network type proof context          |


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

#### default Responses

| Status  | Meaning | Description    | Schema                      |
|:--------|:--------|:---------------|:----------------------------|
| 200     | OK      | Success        | Data : base64 encoded bytes |
| default | Default | JSON-RPC Error | Error Response              |

### btp_getMessages

Get BTP messages.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "btp_getMessages",
  "params": {
    "networkID" : "0x03",
    "height": "0x11"
  }
}
```
#### Parameters

| Name        | Type      | Required  | Description       |
|:------------|:----------|:----------|:------------------|
| height      | T_INT     | true      | Main block height |
| networkID   | T_INT     | true      | BTP network ID    |


> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": [
    "4hCgFOFiPi6RyndLHtrYmLXDDRtgcu6qaC/qJwyoqBc0sT0=",
    "+FCCIAC4S/hJlQE3cjD/tKWh8wO0iMRxpsPFE/2aJ/GYRXZlbnQoQWRkcmVzcyxpbnQsYnl0ZXMplQAxnEr0c2+ow/yTGoPGQBwA==",
    "+HGgdzCygOSkliUIwU6RfeYGyP31o0QgJbVPsCAgICAgICAgA=="
  ]
}
```
#### Responses

| Name   | Type                 | Description                     |
|:-------|:---------------------|:--------------------------------|
| result | T_ARRAY of T_BASE64  | List of base64 encoded messages |

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

#### Default Responses

| Status  | Meaning | Description    | Schema                      |
|:--------|:--------|:---------------|:----------------------------|
| 200     | OK      | Success        | Data : base64 encoded bytes |
| default | Default | JSON-RPC Error | Error Response              |


### btp_getHeader

Get BTP block header

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "btp_getHeader",
  "params": {
    "height": "0x11",
    "networkID" : "0x1"
  }
}
```
#### Parameters

| Name           | Type    | Required | Description       |
|:---------------|:--------|:---------|:------------------|
| height         | T_INT   | true     | Main block height |
| networkID      | T_INT   | true     | Network ID        |


> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": "+QIRoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoCIc9dReCXYR967Ll8MBSUxzksWDY2BnoQi9Wd/7oEoWoPkCx+uBkmGXMdfppwKUS/jaqLBEcxWj4bVoq/WpxFRzoJBir1eJCOvvqV9urYfxHvZ9E4MTcrb9Or7uLXyOQN78oB9ED5ht8egUlm/SGXX1UlpRFz+VwwgN6EY2TH8LJUT7oKsA5iI9WcteAH3ApzQCwO9BGpSHECr7Od0DEGf9/IxAoOsZFmn1IS2/EGAB97IbYRQGIy3j19DS2Y0jWyNmyT5XoERkVHKeInAzSMZcSm22AIIawXF/ibDdskyEDabbdnO5oCxrQAjl/71HrhhG7jokBsviGC3RYglC34NbtOWzZaoHoJMWXQn5I+cRmWg76pmT8VrDO0DSWGMyv1X3GbkPo8w/oPEBG9Q+RjtCMovVi9K6XG08khJpsPtcHB6YkOlHTLa8oPPEZm2q+9Cssdo5l0YzKH7/+cV1h5pxp8baWeUUUssFoBIHc9BwAGJDsArHrh9kkvS6K8B6xmOzRDR0eKfzC9NcoFHqm63YUFSq9I+9gVJB+VDPGWvp6ZV1AejoXwXS/8rkoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoJl4/9qlwu2vrYvpyQ8ayLvfMOd3Tmc3KZT7FTTfJjJ3gA=="
}
```
#### Responses

| Name   | Type      | Description                                      |
|:-------|:----------|:-------------------------------------------------|
| result | T_BASE64  | Base64 encoded [BTPBlockHeader](#btpblockheader) |

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

#### Default Responses

| Status  | Meaning | Description    | Schema                      |
|:--------|:--------|:---------------|:----------------------------|
| 200     | OK      | Success        | Data : base64 encoded bytes |
| default | Default | JSON-RPC Error | Error Response              |

### btp_getProof

Get BTP block proof

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "btp_getProof",
  "params": {
    "height": "0x11",
    "networkID" : "0x1"
  }
}
```
#### Parameters

| Name           | Type    | Required | Description       |
|:---------------|:--------|:---------|:------------------|
| height         | T_INT   | true     | Main block height |
| networkID      | T_INT   | true     | Network ID        |


> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": "+QIRoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoCIc9dReCXYR967Ll8MBSUxzksWDY2BnoQi9Wd/7oEoWoPkCx+uBkmGXMdfppwKUS/jaqLBEcxWj4bVoq/WpxFRzoJBir1eJCOvvqV9urYfxHvZ9E4MTcrb9Or7uLXyOQN78oB9ED5ht8egUlm/SGXX1UlpRFz+VwwgN6EY2TH8LJUT7oKsA5iI9WcteAH3ApzQCwO9BGpSHECr7Od0DEGf9/IxAoOsZFmn1IS2/EGAB97IbYRQGIy3j19DS2Y0jWyNmyT5XoERkVHKeInAzSMZcSm22AIIawXF/ibDdskyEDabbdnO5oCxrQAjl/71HrhhG7jokBsviGC3RYglC34NbtOWzZaoHoJMWXQn5I+cRmWg76pmT8VrDO0DSWGMyv1X3GbkPo8w/oPEBG9Q+RjtCMovVi9K6XG08khJpsPtcHB6YkOlHTLa8oPPEZm2q+9Cssdo5l0YzKH7/+cV1h5pxp8baWeUUUssFoBIHc9BwAGJDsArHrh9kkvS6K8B6xmOzRDR0eKfzC9NcoFHqm63YUFSq9I+9gVJB+VDPGWvp6ZV1AejoXwXS/8rkoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoJl4/9qlwu2vrYvpyQ8ayLvfMOd3Tmc3KZT7FTTfJjJ3gA=="
}
```
#### Responses

| Name   | Type      | Description                 |
|:-------|:----------|:----------------------------|
| result | T_BASE64  | Base64 encoded block proof  |

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

#### Default Responses

| Status  | Meaning | Description    | Schema                      |
|:--------|:--------|:---------------|:----------------------------|
| 200     | OK      | Success        | Data : base64 encoded bytes |
| default | Default | JSON-RPC Error | Error Response              |



### btp_getSourceInformation

Get source network information

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "btp_getSourceInformation"
}
```
#### Parameters
None


> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": {
    "srcNetworkID" : "0x1.icon",
    "networkTypeIDs" : ["0x1","0x2"]
  }

}
```
#### Responses

| Name           | Type           | Description             |
|:---------------|:---------------|:------------------------|
| srcNetworkUID  | T_STRING       | Source network UID      |
| networkTypeIDs | Array of T_INT | List of network type ID |

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

#### Default Responses

| Status  | Meaning | Description    | Schema                      |
|:--------|:--------|:---------------|:----------------------------|
| 200     | OK      | Success        | Data : base64 encoded bytes |
| default | Default | JSON-RPC Error | Error Response              |

## BTPBlockHeader

BTPBlockHeader is `B_LIST` of the following fields

| Name                 | Type       | Comment                                                                                   |
|:---------------------|:-----------|:------------------------------------------------------------------------------------------|
| MainHeight           | B_INT      |                                                                                           |
| Round                | B_INT      |                                                                                           |
| NextProofContextHash | B_BYTES    |                                                                                           |
| NetworkSectionToRoot | B_LIST     | list of MerkleNode for merkle path from H(NetworkSection) to NSRoot                       |
| NetworkID            | B_INT      |                                                                                           |
| UpdateNumber         | B_INT      | See [UpdateNumber](#updatenumber).                                                        |
| Prev                 | B_BYTES(N) | H(NetworkSection) of prev BTP block                                                       |
| MessageCount         | B_INT      |                                                                                           |
| MessagesRoot         | B_BYTES(N) | Merkle root of Messages                                                                   |
| NextProofContext     | B_BYTES(N) | nil if NextProofContextHash is the same as previous block's value. non-nil if Prev is nil |

### MerkleNode

`B_LIST` of the following fields

| Name  | Type       | Comment                                |
|:------|:-----------|:---------------------------------------|
| Dir   | B_INT      | 0 for Left, 1 for Right                |
| Value | B_BYTES(N) | Value for Dir. nil if there is no node |

### MerkleRoot algorithm

```
func MerkleRoot(nodes ...[]byte) []byte {
    l := len(node)
    if l == 1 {
        return nodes[0]
    }
    if l is odd {
        return MerkleRoot(
            Hash(cat(nodes[0], nodes[1])),
            Hash(cat(nodes[2], nodes[3])),
            ...,
            Hash(cat(nodes[l-3], nodes[l-2])),
            node[l-1],
        )
    } else {
        return MerkleRoot(
            Hash(cat(nodes[0], nodes[1])),
            Hash(cat(nodes[2], nodes[3])),
            ...,
            Hash(cat(nodes[l-2], nodes[l-1])),
        )
    }
}
```

### Example Merkle Proof

```
Data : [Hash(1), Hash(2), Hash(3)]
Proof of 3rd node : [[Right, nil], [Left, Hash(cat(Hash(1), Hash(2)))]]
```

### UpdateNumber

```
UpdateNumber = FirstMessageSN << 1 | ProofContextChanged
```

| Name                | Comment                                                                                                                |
|:--------------------|:-----------------------------------------------------------------------------------------------------------------------|
| FirstMessageSN      | Starts from 0. Next message SN of previous NS. See below for example.                                                  |
| ProofContextChanged | 1 if it is the first BTP block for a network or NextProofContextHash is different from the previous BTP block's value. |

### Example of valid FirstMessageSN values

```
NetworkSection0: {
    FirstMessageSN: 0,
    MessageCount: 0,
},
NetworkSection1: {
    FirstMessageSN: 0,
    MessageCount: 5,
    Prev: NetworkSection0.Hash,
},
NetworkSection2: {
    FirstMessageSN: 5,
    MessageCount: 0,
    Prev: NetworkSection1.Hash,
},
NetworkSection3: {
    FirstMessageSN: 5,
    MessageCount: 1,
    Prev: NetworkSection2.Hash,
}
```
