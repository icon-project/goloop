---
title: Node Management API
language_tabs: []
toc_footers: []
includes: []
search: true
highlight_theme: darkula
headingLevel: 2

---

<h1 id="node-management-api">Node Management API v0.1.0</h1>

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

Return system information.

<h3 id="view-system-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|format|query|string|false|Format the output using the given Go template|

> Example responses

> 200 Response

```json
{
  "buildVersion": "v1.3.7",
  "buildTags": "linux/amd64 tags(rocksdb)-2023-05-31-05:27:48",
  "setting": {
    "address": "hx4208599c8f58fed475db747504a80a311a3af63b",
    "p2p": "localhost:8080",
    "p2pListen": "localhost:8080",
    "rpcAddr": ":9080",
    "rpcDump": false
  },
  "config": {
    "eeInstances": 1,
    "rpcBatchLimit": 10,
    "rpcDefaultChannel": "",
    "rpcIncludeDebug": false,
    "rpcRosetta": false,
    "wsMaxSession": 10
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

## View system configuration

<a id="opIdgetSystemConfiguration"></a>

> Code samples

`GET /system/configure`

Return system configuration.

> Example responses

> 200 Response

```json
{
  "eeInstances": 1,
  "rpcBatchLimit": 10,
  "rpcDefaultChannel": "",
  "rpcIncludeDebug": false,
  "rpcRosetta": false,
  "wsMaxSession": 10
}
```

<h3 id="view-system-configuration-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[SystemConfig](#schemasystemconfig)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Configure system

<a id="opIdconfigureSystem"></a>

> Code samples

`POST /system/configure`

Configure system, configurable properties refer to [SystemConfig](#schemasystemconfig)

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
|body|body|[ConfigureParam](#schemaconfigureparam)|true|key-value to configure|

<h3 id="configure-system-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## List Backups

<a id="opIdgetBackups"></a>

> Code samples

`GET /system/backup`

Return list of backups

> Example responses

> 200 Response

```json
[
  {
    "name": "0x178977_0x1_1_20200715-111057.zip",
    "cid": "0x178977",
    "nid": "0x1",
    "channel": "1",
    "height": 2021,
    "codec": "rlp"
  }
]
```

<h3 id="list-backups-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[BackupList](#schemabackuplist)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Restore Status

<a id="opIdgetRestoreStatus"></a>

> Code samples

`GET /system/restore`

View the status of restoring

> Example responses

> 200 Response

```json
{
  "name": "0x178977_0x1_1_20200715-111057.zip",
  "overwrite": true,
  "state": "started 23/128"
}
```

<h3 id="restore-status-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[RestoreStatus](#schemarestorestatus)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Start Restore

<a id="opIdstartRestore"></a>

> Code samples

`POST /system/restore`

Start to restore chain from the backup

> Body parameter

```json
{
  "name": "0x178977_0x1_1_20200715-111057.zip",
  "overwrite": true
}
```

<h3 id="start-restore-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[RestoreParam](#schemarestoreparam)|true|Name of backup and options|

<h3 id="start-restore-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Stop Restore

<a id="opIdstopRestore"></a>

> Code samples

`DELETE /system/restore`

Stop restoring operation

<h3 id="stop-restore-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
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
    "cid": "0x782b03",
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
|» cid|string("0x" + lowercase HEX string)|false|none|chain-id of chain|
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
  patchTxPool: 1000
  maxBlockTxBytes: 1048576
  nodeCache: none
  channel: '000000'
  secureSuites: 'none,tls,ecdhe'
  secureAeads: 'chacha,aes128,aes256'
  defaultWaitTimeout: 0
  txTimeout: 0
  maxWaitTimeout: 0
  autoStart: false
  platform: basic
  childrenLimit: -1
  nephewsLimit: -1
genesisZip: string

```

<h3 id="join-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|object|true|Genesis-Storage zip file and json encoded chain-configuration for join chain using multipart|
|» json|body|[ChainConfig](#schemachainconfig)|true|json encoded chain-configuration, using multipart 'Content-Disposition: name=json'|
|»» dbType|body|string|false|Name of database system, ReadOnly|
|»» seedAddress|body|string|false|List of Seed ip-port, Comma separated string, Runtime-Configurable|
|»» role|body|integer|false|Role:|
|»» concurrencyLevel|body|integer|false|Maximum number of executors to use for concurrency|
|»» normalTxPool|body|integer|false|Size of normal transaction pool|
|»» patchTxPool|body|integer|false|Size of patch transaction pool|
|»» maxBlockTxBytes|body|integer|false|Max size of transactions in a block|
|»» nodeCache|body|string|false|Node cache:|
|»» channel|body|string|false|Chain-alias of node|
|»» secureSuites|body|string|false|Supported Secure suites with order (none,tls,ecdhe) - Comma separated string|
|»» secureAeads|body|string|false|Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string|
|»» defaultWaitTimeout|body|integer|false|Default wait timeout in milli-second(0:disable)|
|»» maxWaitTimeout|body|integer|false|Max wait timeout in milli-second(0:uses same value of defaultWaitTimeout)|
|»» txTimeout|body|integer|false|Transaction timeout in milli-second(0:uses system default value)|
|»» autoStart|body|boolean|false|Start the chain automatically on node start|
|»» platform|body|string|false|Platform to handle transactions(defined by extended software)|
|»» childrenLimit|body|integer|false|Maximum number of child connections(-1: uses system default value)|
|»» nephewsLimit|body|integer|false|Maximum number of nephew connections(-1: uses system default value)|
|»» validateTxOnSend|body|boolean|false|Validate transaction on send(false: no validation)|
|» genesisZip|body|string(binary)|true|Genesis-Storage zip file, using multipart 'Content-Disposition: name=genesisZip'|

#### Detailed descriptions

**»» role**: Role:
 * `0` - None
 * `1` - Seed
 * `2` - Validator
 * `3` - Seed and Validator
Runtime-Configurable

**»» nodeCache**: Node cache:
 * `none` - No cache
 * `small` - Memory Lv1 ~ Lv5 for all
 * `large` - Memory Lv1 ~ Lv5 for all and File Lv6 for store

#### Enumerated Values

|Parameter|Value|
|---|---|
|»» dbType|goleveldb|
|»» dbType|rocksdb|
|»» dbType|mapdb|
|»» role|0|
|»» role|1|
|»» role|2|
|»» role|3|
|»» nodeCache|none|
|»» nodeCache|small|
|»» nodeCache|large|

> Example responses

> 200 Response

<h3 id="join-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[ChainID](#schemachainid)|
|409|[Conflict](https://tools.ietf.org/html/rfc7231#section-6.5.8)|Conflict|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Inspect Chain

<a id="opIdgetChain"></a>

> Code samples

`GET /chain/{cid}`

Return low-level information about a chain.

<h3 id="inspect-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|
|format|query|string|false|Format the output using the given Go template|
|informal|query|boolean|false|Inspect with informal data|

> Example responses

> 200 Response

```json
{
  "cid": "0x782b03",
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
    "patchTxPool": 1000,
    "maxBlockTxBytes": 1048576,
    "nodeCache": "none",
    "channel": "000000",
    "secureSuites": "none,tls,ecdhe",
    "secureAeads": "chacha,aes128,aes256",
    "defaultWaitTimeout": 0,
    "txTimeout": 0,
    "maxWaitTimeout": 0,
    "autoStart": false,
    "platform": "basic",
    "childrenLimit": -1,
    "nephewsLimit": -1
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

`DELETE /chain/{cid}`

Leave Chain.

<h3 id="leave-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|

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

`POST /chain/{cid}/start`

Start Chain.

<h3 id="start-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|

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

`POST /chain/{cid}/stop`

Stop Chain.

<h3 id="stop-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|

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

`POST /chain/{cid}/reset`

Reset Chain.

> Body parameter

```json
{
  "height": 1,
  "blockHash": "0x77ae0f77a345b3e5e8b65f6084cee34d04f037b1b6213134a463781b84006fcc"
}
```

<h3 id="reset-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|
|body|body|[ChainResetParam](#schemachainresetparam)|true|none|

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

`POST /chain/{cid}/import`

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
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|
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

## Prune Chain

<a id="opIdpruneChain"></a>

> Code samples

`POST /chain/{cid}/prune`

Prune chain data from the specific height

> Body parameter

```json
{
  "dbType": "goleveldb",
  "height": 1
}
```

<h3 id="prune-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|
|body|body|[PruneParam](#schemapruneparam)|true|none|

<h3 id="prune-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Backup Chain

<a id="opIdbackupChain"></a>

> Code samples

`POST /chain/{cid}/backup`

Backup chain data to the specific file

> Body parameter

```json
{
  "manual": true
}
```

<h3 id="backup-chain-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|
|body|body|[BackupParam](#schemabackupparam)|false|options for backup|

<h3 id="backup-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Download Genesis-Storage

<a id="opIdgetChainGenesis"></a>

> Code samples

`GET /chain/{cid}/genesis`

Download Genesis-Storage zip file

<h3 id="download-genesis-storage-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|

> Example responses

> 200 Response

<h3 id="download-genesis-storage-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|string|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## View chain configuration

<a id="opIdgetChainConfiguration"></a>

> Code samples

`GET /chain/{cid}/configure`

Return chain configuration.

<h3 id="view-chain-configuration-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|

> Example responses

> 200 Response

```json
{
  "dbType": "goleveldb",
  "seedAddress": "localhost:8080",
  "role": 3,
  "concurrencyLevel": 1,
  "normalTxPool": 5000,
  "patchTxPool": 1000,
  "maxBlockTxBytes": 1048576,
  "nodeCache": "none",
  "channel": "000000",
  "secureSuites": "none,tls,ecdhe",
  "secureAeads": "chacha,aes128,aes256",
  "defaultWaitTimeout": 0,
  "txTimeout": 0,
  "maxWaitTimeout": 0,
  "autoStart": false,
  "platform": "basic",
  "childrenLimit": -1,
  "nephewsLimit": -1
}
```

<h3 id="view-chain-configuration-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[ChainConfig](#schemachainconfig)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

## Configure chain

<a id="opIdconfigureChain"></a>

> Code samples

`POST /chain/{cid}/configure`

Configure chain, configurable properties refer to [ChainConfig](#schemachainconfig)

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
|cid|path|string("0x" + lowercase HEX string)|true|chain-id of chain|
|body|body|[ConfigureParam](#schemaconfigureparam)|true|key-value to configure|

<h3 id="configure-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Not Found|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<aside class="success">
This operation does not require authentication
</aside>

# Schemas

<h2 id="tocSchainid">ChainID</h2>

<a id="schemachainid"></a>

```json
"0x782b03"

```

*chain-id of chain, "0x" + lowercase HEX string*

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|chain-id of chain, "0x" + lowercase HEX string|

<h2 id="tocSchain">Chain</h2>

<a id="schemachain"></a>

```json
{
  "cid": "0x782b03",
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
|cid|string("0x" + lowercase HEX string)|false|none|chain-id of chain|
|nid|string("0x" + lowercase HEX string)|false|none|network-id of chain|
|channel|string|false|none|chain-alias of node|
|height|integer(int64)|false|none|block height of chain|
|state|string|false|none|state of chain|
|lastError|string|false|none|last error of chain|

<h2 id="tocSchaininspect">ChainInspect</h2>

<a id="schemachaininspect"></a>

```json
{
  "cid": "0x782b03",
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
    "patchTxPool": 1000,
    "maxBlockTxBytes": 1048576,
    "nodeCache": "none",
    "channel": "000000",
    "secureSuites": "none,tls,ecdhe",
    "secureAeads": "chacha,aes128,aes256",
    "defaultWaitTimeout": 0,
    "txTimeout": 0,
    "maxWaitTimeout": 0,
    "autoStart": false,
    "platform": "basic",
    "childrenLimit": -1,
    "nephewsLimit": -1
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
  "patchTxPool": 1000,
  "maxBlockTxBytes": 1048576,
  "nodeCache": "none",
  "channel": "000000",
  "secureSuites": "none,tls,ecdhe",
  "secureAeads": "chacha,aes128,aes256",
  "defaultWaitTimeout": 0,
  "txTimeout": 0,
  "maxWaitTimeout": 0,
  "autoStart": false,
  "platform": "basic",
  "childrenLimit": -1,
  "nephewsLimit": -1
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|dbType|string|false|none|Name of database system, ReadOnly|
|seedAddress|string|false|none|List of Seed ip-port, Comma separated string, Runtime-Configurable|
|role|integer|false|none|Role:  * `0` - None  * `1` - Seed  * `2` - Validator  * `3` - Seed and Validator Runtime-Configurable|
|concurrencyLevel|integer|false|none|Maximum number of executors to use for concurrency|
|normalTxPool|integer|false|none|Size of normal transaction pool|
|patchTxPool|integer|false|none|Size of patch transaction pool|
|maxBlockTxBytes|integer|false|none|Max size of transactions in a block|
|nodeCache|string|false|none|Node cache:  * `none` - No cache  * `small` - Memory Lv1 ~ Lv5 for all  * `large` - Memory Lv1 ~ Lv5 for all and File Lv6 for store|
|channel|string|false|none|Chain-alias of node|
|secureSuites|string|false|none|Supported Secure suites with order (none,tls,ecdhe) - Comma separated string|
|secureAeads|string|false|none|Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string|
|defaultWaitTimeout|integer|false|none|Default wait timeout in milli-second(0:disable)|
|maxWaitTimeout|integer|false|none|Max wait timeout in milli-second(0:uses same value of defaultWaitTimeout)|
|txTimeout|integer|false|none|Transaction timeout in milli-second(0:uses system default value)|
|autoStart|boolean|false|none|Start the chain automatically on node start|
|platform|string|false|none|Platform to handle transactions(defined by extended software)|
|childrenLimit|integer|false|none|Maximum number of child connections(-1: uses system default value)|
|nephewsLimit|integer|false|none|Maximum number of nephew connections(-1: uses system default value)|
|validateTxOnSend|boolean|false|none|Validate transaction on send(false: no validation)|

#### Enumerated Values

|Property|Value|
|---|---|
|dbType|goleveldb|
|dbType|rocksdb|
|dbType|mapdb|
|role|0|
|role|1|
|role|2|
|role|3|
|nodeCache|none|
|nodeCache|small|
|nodeCache|large|

<h2 id="tocSchainresetparam">ChainResetParam</h2>

<a id="schemachainresetparam"></a>

```json
{
  "height": 1,
  "blockHash": "0x77ae0f77a345b3e5e8b65f6084cee34d04f037b1b6213134a463781b84006fcc"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|height|int64|false|none|Block Height|
|blockHash|string("0x" + lowercase HEX string)|false|none|Block Hash|

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
  "buildVersion": "v1.3.7",
  "buildTags": "linux/amd64 tags(rocksdb)-2023-05-31-05:27:48",
  "setting": {
    "address": "hx4208599c8f58fed475db747504a80a311a3af63b",
    "p2p": "localhost:8080",
    "p2pListen": "localhost:8080",
    "rpcAddr": ":9080",
    "rpcDump": false
  },
  "config": {
    "eeInstances": 1,
    "rpcBatchLimit": 10,
    "rpcDefaultChannel": "",
    "rpcIncludeDebug": false,
    "rpcRosetta": false,
    "wsMaxSession": 10
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
|» rpcAddr|string|false|none|Listen ip-port of JSON-RPC|
|» rpcDump|boolean|false|none|JSON-RPC Request, Response Dump flag|
|config|[SystemConfig](#schemasystemconfig)|false|none|none|

<h2 id="tocSsystemconfig">SystemConfig</h2>

<a id="schemasystemconfig"></a>

```json
{
  "eeInstances": 1,
  "rpcBatchLimit": 10,
  "rpcDefaultChannel": "",
  "rpcIncludeDebug": false,
  "rpcRosetta": false,
  "wsMaxSession": 10
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|eeInstances|integer|false|none|Number of execution engines|
|rpcBatchLimit|integer|false|none|JSON-RPC batch limit|
|rpcDefaultChannel|string|false|none|default channel for legacy api|
|rpcIncludeDebug|boolean|false|none|Enable JSON-RPC for debug APIs|
|rpcRosetta|boolean|false|none|Enable JSON-RPC for Rosetta|
|wsMaxSession|integer|false|none|Websocket session limit|

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

<h2 id="tocSpruneparam">PruneParam</h2>

<a id="schemapruneparam"></a>

```json
{
  "dbType": "goleveldb",
  "height": 1
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|dbType|string|false|none|Database type|
|height|int64|true|none|Block Height|

<h2 id="tocSbackupparam">BackupParam</h2>

<a id="schemabackupparam"></a>

```json
{
  "manual": true
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|manual|boolean|false|none|Manual backup|

<h2 id="tocSbackuplist">BackupList</h2>

<a id="schemabackuplist"></a>

```json
[
  {
    "name": "0x178977_0x1_1_20200715-111057.zip",
    "cid": "0x178977",
    "nid": "0x1",
    "channel": "1",
    "height": 2021,
    "codec": "rlp"
  }
]

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|false|none|name of the backup|
|cid|string("0x" + lowercase HEX string)|false|none|chain-id of chain|
|nid|string|false|none|Network ID of the backup|
|height|integer|false|none|Last block height of the backup|
|size|integer|false|none|Size of the backup in bytes|
|codec|string|false|none|codec name|

<h2 id="tocSrestorestatus">RestoreStatus</h2>

<a id="schemarestorestatus"></a>

```json
{
  "name": "0x178977_0x1_1_20200715-111057.zip",
  "overwrite": true,
  "state": "started 23/128"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|state|string|true|none|State of the job (stopped, started N/T, stopping, failed, success)|
|name|string|false|none|Name of backup|
|overwrite|boolean|false|none|Whether it replaces existing chain data|

<h2 id="tocSrestoreparam">RestoreParam</h2>

<a id="schemarestoreparam"></a>

```json
{
  "name": "0x178977_0x1_1_20200715-111057.zip",
  "overwrite": true
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|Name of the backup to restore|
|overwrite|boolean|false|none|Whether it replaces existing chain|

