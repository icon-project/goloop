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

## View system

<a id="opIdgetSystem"></a>

> Code samples

`GET /system`

Return System Infomation.

<h3 id="view-system-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|format|query|string|false|Format the output using the given Go template|

> Example responses

> 200 Response

```json
{
  "buildVersion": "v0.1.7",
  "buildTags": "linux/amd64 tags()-2019-08-20-09:39:15",
  "setting": {
    "address": "hx4208599c8f58fed475db747504a80a311a3af63b",
    "p2p": "localhost:8080",
    "p2pListen": "localhost:8080",
    "rpcAddr": ":9080",
    "rpcDump": false
  },
  "config": {
    "eeInstances": 1,
    "rpcDefaultChannel": "",
    "rpcIncludeDebug": false
  }
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
    "state": "started",
    "height": 100,
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
  dbType: goleveldb
  seedAddress: 'localhost:8080'
  role: 3
  concurrencyLevel: 1
  normalTxPool: 5000
  patchTxPool: 5000
  maxBlockTxBytes: 1048576
  channel: '000000'
  secureSuites: 'none,tls,ecdhe'
  secureAeads: 'chacha,aes128,aes256'
genesisZip: string

```

<h3 id="join-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|object|true|Genesis-Storage zip file and json encoded chain-configuration for join chain using multipart|
|» json|body|[ChainConfig](#schemachainconfig)|true|json encoded chain-configuration, using multipart 'Content-Disposition: name=json'|
|»» dbType|body|string|false|Name of database system|
|»» seedAddress|body|string|false|List of Seed ip-port, Comma separated string|
|»» role|body|integer|false|Role:|
|»» concurrencyLevel|body|integer|false|Maximum number of executors to use for concurrency|
|»» normalTxPool|body|integer|false|Size of normal transaction pool|
|»» patchTxPool|body|integer|false|Size of patch transaction pool|
|»» maxBlockTxBytes|body|integer|false|Max size of transactions in a block|
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
|»» dbType|badgerdb|
|»» dbType|goleveldb|
|»» dbType|boltdb|
|»» dbType|mapdb|
|»» role|0|
|»» role|1|
|»» role|2|
|»» role|3|

> Example responses

> 200 Response

<h3 id="join-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[NetworkID](#schemanetworkid)|
|409|[Conflict](https://tools.ietf.org/html/rfc7231#section-6.5.8)|Conflict|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Inspect Chain

<a id="opIdgetChain"></a>

> Code samples

`GET /chain/{nid}`

Return low-level information about a chain.

<h3 id="inspect-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|
|format|query|string|false|Format the output using the given Go template|

> Example responses

> 200 Response

```json
{
  "nid": "0x000000",
  "channel": "000000",
  "state": "started",
  "height": 100,
  "lastError": "",
  "genesisTx": {},
  "config": {
    "dbType": "goleveldb",
    "seedAddress": "localhost:8080",
    "role": 3,
    "concurrencyLevel": 1,
    "normalTxPool": 5000,
    "patchTxPool": 5000,
    "maxBlockTxBytes": 1048576,
    "channel": "000000",
    "secureSuites": "none,tls,ecdhe",
    "secureAeads": "chacha,aes128,aes256"
  },
  "module": {
    "property1": {},
    "property2": {}
  }
}
```

<h3 id="inspect-chain-responses">Responses</h3>

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

## Reset Chain

<a id="opIdresetChain"></a>

> Code samples

`POST /chain/{nid}/reset`

Reset Chain.

<h3 id="reset-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|

<h3 id="reset-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Import Chain

<a id="opIdimportChain"></a>

> Code samples

`POST /chain/{nid}/import`

Import a chain from legacy database.

> Body parameter

```json
{
  "dbPath": "/path/to/database",
  "height": 1
}
```

<h3 id="import-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|
|body|body|[ChainImportParam](#schemachainimportparam)|true|none|

<h3 id="import-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Configure chain

<a id="opIdconfigureChain"></a>

> Code samples

`POST /chain/{nid}/configure`

Configure chain.

> Body parameter

```json
{
  "key": "string",
  "value": "string"
}
```

<h3 id="configure-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|
|body|body|[ConfigureParam](#schemaconfigureparam)|true|refer ChainConfig properties|

<h3 id="configure-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Configure system

<a id="opIdconfigureSystem"></a>

> Code samples

`POST /system/configure`

Configure system

> Body parameter

```json
{
  "key": "string",
  "value": "string"
}
```

<h3 id="configure-system-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|nid|path|string("0x" + lowercase HEX string)|true|network-id of chain|
|body|body|[ConfigureParam](#schemaconfigureparam)|true|refer SystemConfig properties|

<h3 id="configure-system-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

# Schemas

<h2 id="tocSnetworkid">NetworkID</h2>

<a id="schemanetworkid"></a>

```json
"0x000000"

```

*network-id of chain, "0x" + lowercase HEX string*

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|network-id of chain, "0x" + lowercase HEX string|

<h2 id="tocSchain">Chain</h2>

<a id="schemachain"></a>

```json
{
  "nid": "0x000000",
  "channel": "000000",
  "state": "started",
  "height": 100,
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
  "state": "started",
  "height": 100,
  "lastError": "",
  "genesisTx": {},
  "config": {
    "dbType": "goleveldb",
    "seedAddress": "localhost:8080",
    "role": 3,
    "concurrencyLevel": 1,
    "normalTxPool": 5000,
    "patchTxPool": 5000,
    "maxBlockTxBytes": 1048576,
    "channel": "000000",
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
|» genesisTx|object|false|none|Genesis Transaction|
|» config|[ChainConfig](#schemachainconfig)|false|none|none|
|» module|object|false|none|none|
|»» **additionalProperties**|object|false|none|none|

<h2 id="tocSchainconfig">ChainConfig</h2>

<a id="schemachainconfig"></a>

```json
{
  "dbType": "goleveldb",
  "seedAddress": "localhost:8080",
  "role": 3,
  "concurrencyLevel": 1,
  "normalTxPool": 5000,
  "patchTxPool": 5000,
  "maxBlockTxBytes": 1048576,
  "channel": "000000",
  "secureSuites": "none,tls,ecdhe",
  "secureAeads": "chacha,aes128,aes256"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|dbType|string|false|none|Name of database system|
|seedAddress|string|false|none|List of Seed ip-port, Comma separated string|
|role|integer|false|none|Role:  * `0` - None  * `1` - Seed  * `2` - Validator  * `3` - Seed and Validator|
|concurrencyLevel|integer|false|none|Maximum number of executors to use for concurrency|
|normalTxPool|integer|false|none|Size of normal transaction pool|
|patchTxPool|integer|false|none|Size of patch transaction pool|
|maxBlockTxBytes|integer|false|none|Max size of transactions in a block|
|channel|string|false|none|Chain-alias of node|
|secureSuites|string|false|none|Supported Secure suites with order (none,tls,ecdhe) - Comma separated string|
|secureAeads|string|false|none|Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string|

#### Enumerated Values

|Property|Value|
|---|---|
|dbType|badgerdb|
|dbType|goleveldb|
|dbType|boltdb|
|dbType|mapdb|
|role|0|
|role|1|
|role|2|
|role|3|

<h2 id="tocSchainimportparam">ChainImportParam</h2>

<a id="schemachainimportparam"></a>

```json
{
  "dbPath": "/path/to/database",
  "height": 1
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|dbPath|string|true|none|Database path|
|height|int64|true|none|Block Height|

<h2 id="tocSsystem">System</h2>

<a id="schemasystem"></a>

```json
{
  "buildVersion": "v0.1.7",
  "buildTags": "linux/amd64 tags()-2019-08-20-09:39:15",
  "setting": {
    "address": "hx4208599c8f58fed475db747504a80a311a3af63b",
    "p2p": "localhost:8080",
    "p2pListen": "localhost:8080",
    "rpcAddr": ":9080",
    "rpcDump": false
  },
  "config": {
    "eeInstances": 1,
    "rpcDefaultChannel": "",
    "rpcIncludeDebug": false
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|buildVersion|string|false|none|build version|
|buildTags|string|false|none|buildTags|
|setting|object|false|none|none|
|» address|string|false|none|wallet address|
|» p2p|string|false|none|p2p address|
|» p2pListen|string|false|none|p2p listen address|
|config|[SystemConfig](#schemasystemconfig)|false|none|none|

<h2 id="tocSsystemconfig">SystemConfig</h2>

<a id="schemasystemconfig"></a>

```json
{
  "eeInstances": 1,
  "rpcDefaultChannel": "",
  "rpcIncludeDebug": false
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|eeInstances|integer|false|none|eeInstances|
|rpcDefaultChannel|string|false|none|default channel for legacy api|
|rpcIncludeDebug|boolean|false|none|JSON-RPC Response with detail information|

<h2 id="tocSconfigureparam">ConfigureParam</h2>

<a id="schemaconfigureparam"></a>

```json
{
  "key": "string",
  "value": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|key|string|true|none|configuration field name|
|value|string|true|none|configuration value|

