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
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
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
    "nid": 1,
    "height": 100,
    "state": "started"
  }
]
```

<h3 id="list-chains-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|Inline|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|Internal Server Error|None|

<h3 id="list-chains-responseschema">Response Schema</h3>

Status Code **200**

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[[Chain](#schemachain)]|false|none|none|
|» nid|integer(int64)|false|none|nid|
|» height|integer(int64)|false|none|height|
|» state|string|false|none|state|

<aside class="success">
This operation does not require authentication
</aside>

## Join Chain

<a id="opIdjoinChain"></a>

> Code samples

`POST /chain`

Join Chain

<h3 id="join-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|None|
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
|nid|path|integer(int64)|true|Chain Network ID|

> Example responses

> 200 Response

```json
{
  "nid": 1,
  "height": 100,
  "state": "started"
}
```

<h3 id="get-chain-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Success|[Chain](#schemachain)|
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
|nid|path|integer(int64)|true|Chain Network ID|

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
|nid|path|integer(int64)|true|Chain Network ID|

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
|nid|path|integer(int64)|true|Chain Network ID|

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
  "nid": 1,
  "height": 100,
  "state": "started"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|nid|integer(int64)|false|none|nid|
|height|integer(int64)|false|none|height|
|state|string|false|none|state|

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
|buildVersion|string|false|none|buildVersion|
|buildTags|string|false|none|buildTags|
|address|string|false|none|address|
|p2p|string|false|none|p2p address|
|p2pListen|string|false|none|p2p listen address|

