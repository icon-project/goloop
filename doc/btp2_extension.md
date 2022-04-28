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
  "networkType" : "0x2",
  "networkId" : "0x1"
}
```
#### Parameters

| Name        | Type     | Required    | Description              |
|:------------|:---------|:------------|:-------------------------|
| height      | T_INT    | true        | Start height             |
| networkType | T_INT    | true        | Destination network type |
| networkId   | T_INT    | true        | Destination network ID   |
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

| Name   | Type   | Description                  |
|:-------|:-------|:-----------------------------|
| header | T_SIG  | Base64 encoded BTP Header    |

## BTP JSON-RPC Methods

### btp_getNetworkInformation

Get network id information.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "btp_getNetworkInformation",
  "params": {
    "networkId" : "0x3"
  }
}
```
#### Parameters

| Name      | Type    | Required | Description            |
|:----------|:--------|:---------|:-----------------------|
| height    | T_INT   | false    | Block height           |
| networkId | T_INT   | true     | Destination network id |


> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": {
    "startHeight" : "0x11",
    "networkTypeId" : "0x1",
    "networkTypeName" : "eth",
    "networkId" : "0x3",
    "lastMessagesRootNumber" : "0x20",
    "prevNSHash" : "0x…",
    "lastNSHash" : "0x…"
  }
}
```
#### Responses

| Name                  | Type      | Description                               |
|:----------------------|:----------|:------------------------------------------|
| startHeight           | T_INT     | Block height where btp message started    |
| networkTypeId         | T_INT     | Network type id                           |
| networkTypeName       | T_STRING  | Network type name                         |
| networkId             | T_INT     | Network id                                |
| lastMessageRootNumber | T_INT     | MessageRootSN and UpdateFlag              |
| prevNSHash            | T_HASH    | Previous network hash                     |
| lastNSHash            | T_HASH    | Last network hash                         |

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


### btp_getNetworkType

Get network types information.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "icx_getNetworkType",
  "params": {
    "height" : "0x11",
    "networkTypeId" : "0x02"
  }
}
```
#### Parameters

| Name          | Type    | Required | Description     |
|:--------------|:--------|:---------|:----------------|
| height        | T_INT   | true     | Block height    |
| networkTypeId | T_INT   | true     | Network type ID |



> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": {
    "networkTypeId" : "0x2",
    "networkTypeName" : "eth",
    "activeNetworkType" : "0x1",
    "connectedNetworks" : ["0x3","0x4"],
    "proofContext" : "+QIRoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoCIc9dReCXYR967Ll8MBSUxzksWDY2BnoQi9Wd/7oEoWoPkCx+uBkmGXMdfppwKUS/jaqLBEcxWj4bVoq/WpxFRzoJBir1eJCOvvqV9urYfxHvZ9E4MTcrb9Or7uLXyOQN78oB9ED5ht8egUlm/SGXX1UlpRFz+VwwgN6EY2TH8LJUT7oKsA5iI9WcteAH3ApzQCwO9BGpSHECr7Od0DEGf9/IxAoOsZFmn1IS2/EGAB97IbYRQGIy3j19DS2Y0jWyNmyT5XoERkVHKeInAzSMZcSm22AIIawXF/ibDdskyEDabbdnO5oCxrQAjl/71HrhhG7jokBsviGC3RYglC34NbtOWzZaoHoJMWXQn5I+cRmWg76pmT8VrDO0DSWGMyv1X3GbkPo8w/oPEBG9Q+RjtCMovVi9K6XG08khJpsPtcHB6YkOlHTLa8oPPEZm2q+9Cssdo5l0YzKH7/+cV1h5pxp8baWeUUUssFoBIHc9BwAGJDsArHrh9kkvS6K8B6xmOzRDR0eKfzC9NcoFHqm63YUFSq9I+9gVJB+VDPGWvp6ZV1AejoXwXS/8rkoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoJl4/9qlwu2vrYvpyQ8ayLvfMOd3Tmc3KZT7FTTfJjJ3gA=="
  }
}
```
#### Responses

| Name                | Type              | Description                         |
|:--------------------|:------------------|:------------------------------------|
| networkTypeId       | T_INT             | Network type id                     |
| networkTypeName     | T_STRING          | Network type name                   |
| activeNetworkType   | T_INT             | Active state of network type        |
| connectedNetworks   | T_ARRAY of T_INT  | Network id included in network type |
| proofContext        | T_BYTES           | Network type proof context          |


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

Get BTP block messages.

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "btp_getMessages",
  "params": {
    "networkTypeId" : "0x02",
    "networkId" : "0x03",
    "height": "0x11"
  }
}
```
#### Parameters

| Name        | Type      | Required  | Description              |
|:------------|:----------|:----------|:-------------------------|
| networkType | T_INT     | true      | Destination network type |
| networkId   | T_INT     | true      | Destination network ID   |
| height      | T_INT     | true      | Block height             |


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

| Name   | Type             | Description                     |
|:-------|:-----------------|:--------------------------------|
| result | T_ARRAY of T_SIG | List of base64 encoded messages |

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

Get btp block header

> Request

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "method": "btp_getHeader",
  "params": {
    "height": "0x11",
    "networkTypeId" : "0x2",
    "networkId" : "0x1"
  }
}
```
#### Parameters

| Name           | Type    | Required | Description                 |
|:---------------|:--------|:---------|:----------------------------|
| height         | T_INT   | true     | Block height                |
| networkTypeId  | T_INT   | true     | Destination network type ID |
| networkId      | T_INT   | true     | Destination network ID      |


> Sample responses

```json
{
  "id": 1001,
  "jsonrpc": "2.0",
  "result": "+QIRoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoCIc9dReCXYR967Ll8MBSUxzksWDY2BnoQi9Wd/7oEoWoPkCx+uBkmGXMdfppwKUS/jaqLBEcxWj4bVoq/WpxFRzoJBir1eJCOvvqV9urYfxHvZ9E4MTcrb9Or7uLXyOQN78oB9ED5ht8egUlm/SGXX1UlpRFz+VwwgN6EY2TH8LJUT7oKsA5iI9WcteAH3ApzQCwO9BGpSHECr7Od0DEGf9/IxAoOsZFmn1IS2/EGAB97IbYRQGIy3j19DS2Y0jWyNmyT5XoERkVHKeInAzSMZcSm22AIIawXF/ibDdskyEDabbdnO5oCxrQAjl/71HrhhG7jokBsviGC3RYglC34NbtOWzZaoHoJMWXQn5I+cRmWg76pmT8VrDO0DSWGMyv1X3GbkPo8w/oPEBG9Q+RjtCMovVi9K6XG08khJpsPtcHB6YkOlHTLa8oPPEZm2q+9Cssdo5l0YzKH7/+cV1h5pxp8baWeUUUssFoBIHc9BwAGJDsArHrh9kkvS6K8B6xmOzRDR0eKfzC9NcoFHqm63YUFSq9I+9gVJB+VDPGWvp6ZV1AejoXwXS/8rkoJM2lLiv1hugUrj98X/c2Q8IWwOOjY5X5hoXhJWxYt9HoJl4/9qlwu2vrYvpyQ8ayLvfMOd3Tmc3KZT7FTTfJjJ3gA=="
}
```
#### Responses

| Name   | Type      | Description                |
|:-------|:----------|:---------------------------|
| result | T_SIG     | Base64 encoded BTP Header  |

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
    "srcNetworkId" : "btp://0x1.icon/…",
    "networkTypeIds" : ["0x1","0x2"]
  }

}
```
#### Responses

| Name           | Type      | Description             |
|:---------------|:----------|:------------------------|
| srcNetworkId   | T_STRING  | Source network id       |
| networkTypeIds | T_SIG     | List of network type id |

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
