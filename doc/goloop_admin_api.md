---
title: Node Management API
language_tabs: []
toc_footers: []
includes: []
search: true
highlight_theme: darkula
headingLevel: 2

---

<h1 id="node-management-api">Node Management API v0.0.1</h1>

> Scroll down for example requests and responses.

goloop management

Base URLs:

* <a href="http://localhost:9080/admin">http://localhost:9080/admin</a>

<h1 id="node-management-api-node">node</h1>

Node Management

## View System

<a id="opIdgetSystem"></a>

> Code samples

`GET /system`

Return System Infomation.

> Example responses

> 200 Response

```json
{
  "buildVersion": "release-gs/v0.1.1-2",
  "buildTags": "darwin/amd64 tags()-2019-05-31-13:51:28",
  "address": "hx4208599c8f58fed475db747504a80a311a3af63b",
  "p2p": "localhost:8080",
  "p2pListen": "localhost:8080"
}
```

<h3 id="view-system-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[System](#schemasystem)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

<h1 id="node-management-api-chain">chain</h1>

Chain Management

## List Chains

<a id="opIdlistChain"></a>

> Code samples

`GET /chain`

Returns a list of chains

> Example responses

> 200 Response

```json
[
  {
    "nid": "0x000000",
    "channel": "000000",
    "height": 100,
    "state": "started",
    "lastError": ""
  }
]
```

<h3 id="list-chains-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|Inline|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<h3 id="list-chains-responseschema">Response Schema</h3>

Status Code **200**

*array of chains*

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[[Chain](#schemachain)]|false|none|array of chains|
|» nid|string("0x" + lowercase HEX string)|false|none|network-id of chain|
|» channel|string|false|none|chain-alias of node|
|» height|integer(int64)|false|none|block height of chain|
|» state|string|false|none|state of chain|
|» lastError|string|false|none|last error of chain|

<aside class="success">
This operation does not require authentication
</aside>

## Join Chain

<a id="opIdjoinChain"></a>

> Code samples

`POST /chain`

Join Chain

> Body parameter

```yaml
json:
  db_type: goleveldb
  seed_addr: string
  role: 3
  concurrency_level: 1
  normal_tx_pool: 1
  patch_tx_pool: 1
  max_block_tx_bytes: 1
  channel: ''
  secureSuites: 'none,tls,ecdhe'
  secureAeads: 'chacha,aes128,aes256'
genesisZip: string

```

<h3 id="join-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|object|true|Genesis-Storage zip file and json encoded chain-configuration for join chain using multipart|
|» json|body|[ChainConfig](#schemachainconfig)|true|json encoded chain-configuration, using multipart 'Content-Disposition: name=json'|
|»» db_type|body|string|false|Name of database system|
|»» seed_addr|body|string|false|Ip-port of Seed|
|»» role|body|integer|false|Role:|
|»» concurrency_level|body|integer|false|Maximum number of executors to use for concurrency|
|»» normal_tx_pool|body|integer|false|Maximum number of executors to use for concurrency|
|»» patch_tx_pool|body|integer|false|Maximum number of executors to use for concurrency|
|»» max_block_tx_bytes|body|integer|false|Maximum number of executors to use for concurrency|
|»» channel|body|string|false|Chain-alias of node|
|»» secureSuites|body|string|false|Supported Secure suites with order (none,tls,ecdhe) - Comma separated string|
|»» secureAeads|body|string|false|Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string|
|» genesisZip|body|string(binary)|true|Genesis-Storage zip file, using multipart 'Content-Disposition: name=genesisZip'|

#### Detailed descriptions

**»» role**: Role:
 * `0` - None
 * `1` - Seed
 * `2` - Validator
 * `3` - Seed and Validator

#### Enumerated Values

|Parameter|Value|
|---|---|
|»» db_type|badgerdb|
|»» db_type|goleveldb|
|»» db_type|boltdb|
|»» db_type|mapdb|
|»» role|0|
|»» role|1|
|»» role|2|
|»» role|3|

<h3 id="join-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|409|[Conflict](https://tools.ietf.org/html/rfc7231#section-6.5.8)|Conflict|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Get Chain

<a id="opIdgetChain"></a>

> Code samples

`GET /chain/{nid}`

Get Chain Infomation.

<h3 id="get-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|

> Example responses

> 200 Response

```json
{
  "nid": "0x000000",
  "channel": "000000",
  "height": 100,
  "state": "started",
  "lastError": "",
  "genesisTx": {},
  "config": {
    "db_type": "goleveldb",
    "seed_addr": "string",
    "role": 3,
    "concurrency_level": 1,
    "normal_tx_pool": 1,
    "patch_tx_pool": 1,
    "max_block_tx_bytes": 1,
    "channel": "",
    "secureSuites": "none,tls,ecdhe",
    "secureAeads": "chacha,aes128,aes256"
  },
  "module": {
    "property1": {},
    "property2": {}
  }
}
```

<h3 id="get-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[ChainInspect](#schemachaininspect)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Leave Chain

<a id="opIdleaveChain"></a>

> Code samples

`DELETE /chain/{nid}`

Leave Chain.

<h3 id="leave-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|

<h3 id="leave-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Start Chain

<a id="opIdstartChain"></a>

> Code samples

`POST /chain/{nid}/start`

Start Chain.

<h3 id="start-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|

<h3 id="start-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Stop Chain

<a id="opIdstopChain"></a>

> Code samples

`POST /chain/{nid}/stop`

Stop Chain.

<h3 id="stop-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|

<h3 id="stop-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

# Schemas

<h2 id="tocSchain">Chain</h2>

<a id="schemachain"></a>

```json
{
  "nid": "0x000000",
  "channel": "000000",
  "height": 100,
  "state": "started",
  "lastError": ""
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|nid|string("0x" + lowercase HEX string)|false|none|network-id of chain|
|channel|string|false|none|chain-alias of node|
|height|integer(int64)|false|none|block height of chain|
|state|string|false|none|state of chain|
|lastError|string|false|none|last error of chain|

<h2 id="tocSchaininspect">ChainInspect</h2>

<a id="schemachaininspect"></a>

```json
{
  "nid": "0x000000",
  "channel": "000000",
  "height": 100,
  "state": "started",
  "lastError": "",
  "genesisTx": {},
  "config": {
    "db_type": "goleveldb",
    "seed_addr": "string",
    "role": 3,
    "concurrency_level": 1,
    "normal_tx_pool": 1,
    "patch_tx_pool": 1,
    "max_block_tx_bytes": 1,
    "channel": "",
    "secureSuites": "none,tls,ecdhe",
    "secureAeads": "chacha,aes128,aes256"
  },
  "module": {
    "property1": {},
    "property2": {}
  }
}

```

### Properties

*allOf*

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[Chain](#schemachain)|false|none|none|

*and*

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|object|false|none|none|
|» genesisTx|object|false|none|none|
|» config|[ChainConfig](#schemachainconfig)|false|none|none|
|» module|object|false|none|none|
|»» **additionalProperties**|object|false|none|none|

<h2 id="tocSchainconfig">ChainConfig</h2>

<a id="schemachainconfig"></a>

```json
{
  "db_type": "goleveldb",
  "seed_addr": "string",
  "role": 3,
  "concurrency_level": 1,
  "normal_tx_pool": 1,
  "patch_tx_pool": 1,
  "max_block_tx_bytes": 1,
  "channel": "",
  "secureSuites": "none,tls,ecdhe",
  "secureAeads": "chacha,aes128,aes256"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|db_type|string|false|none|Name of database system|
|seed_addr|string|false|none|Ip-port of Seed|
|role|integer|false|none|Role:  * `0` - None  * `1` - Seed  * `2` - Validator  * `3` - Seed and Validator|
|concurrency_level|integer|false|none|Maximum number of executors to use for concurrency|
|normal_tx_pool|integer|false|none|Maximum number of executors to use for concurrency|
|patch_tx_pool|integer|false|none|Maximum number of executors to use for concurrency|
|max_block_tx_bytes|integer|false|none|Maximum number of executors to use for concurrency|
|channel|string|false|none|Chain-alias of node|
|secureSuites|string|false|none|Supported Secure suites with order (none,tls,ecdhe) - Comma separated string|
|secureAeads|string|false|none|Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string|

#### Enumerated Values

|Property|Value|
|---|---|
|db_type|badgerdb|
|db_type|goleveldb|
|db_type|boltdb|
|db_type|mapdb|
|role|0|
|role|1|
|role|2|
|role|3|

<h2 id="tocSsystem">System</h2>

<a id="schemasystem"></a>

```json
{
  "buildVersion": "release-gs/v0.1.1-2",
  "buildTags": "darwin/amd64 tags()-2019-05-31-13:51:28",
  "address": "hx4208599c8f58fed475db747504a80a311a3af63b",
  "p2p": "localhost:8080",
  "p2pListen": "localhost:8080"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|buildVersion|string|false|none|build version|
|buildTags|string|false|none|buildTags|
|address|string|false|none|wallet address|
|p2p|string|false|none|p2p address|
|p2pListen|string|false|none|p2p listen address|

