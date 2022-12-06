# ICON Chain SCORE API

- [IISS](#iiss)
  * ReadOnly APIs
    + [getStake](#getstake)
    + [getDelegation](#getdelegation)
    + [getBond](#getbond)
    + [queryIScore](#queryiscore)
    + [getPRep](#getprep)
    + [getPReps](#getpreps)
    + [getBonderList](#getbonderlist)
  * Writable APIs
    + [setStake](#setstake)
    + [setDelegation](#setdelegation)
    + [setBond](#setbond)
    + [claimIScore](#claimiscore)
    + [registerPRep](#registerprep)
    + [setPRep](#setprep)
    + [unregisterPRep](#unregisterprep)
    + [setBonderList](#setbonderlist)
- [BTP](#btp)
  * ReadOnly APIs
    + [getBTPNetworkTypeID](#getbtpnetworktypeid)
    + [getPRepNodePublicKey](#getprepnodepublickey)
  * Writable APIs
    + [openBTPNetwork](#openbtpnetwork)
    + [closeBTPNetwork](#closebtpnetwork)
    + [sendBTPMessage](#sendbtpmessage)
    + [registerPRepNodePublicKey](#registerprepnodepublickey)
    + [setPRepNodePublicKey](#setprepnodepublickey)
- [Types](#types)
  * [Unstake](#unstake)
  * [Vote](#vote)
  * [Unbond](#unbond)
  * [PRep](#prep)

# IISS

## ReadOnly APIs

### getStake

Returns the stake status of a given `address`.

```python
def getStake(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description      |
|:--------|:--------|:-----------------|
| address | Address | address to query |

*Returns:*

| Key      | Value Type                  | Description                 |
|:---------|:----------------------------|:----------------------------|
| stake    | int                         | ICX amount of stake in loop |
| unstakes | List\[[Unstake](#unstake)\] | List of Unstake information |

*Revision:* 5 ~ 

### getDelegation

Returns the delegation status of a given `address`.

```python
def getDelegation(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description      |
|:--------|:--------|:-----------------|
| address | Address | address to query |

*Returns:*

| Key            | Value type            | Description                                                                  |
|:---------------|:----------------------|:-----------------------------------------------------------------------------|
| totalDelegated | int                   | The sum of delegation amount                                                 |
| votingPower    | int                   | Remaining amount of stake that ICONist can delegate and bond to other P-Reps |
| delegations    | List\[[Vote](#vote)\] | List of delegation information (MAX: 100 entries)                            |

*Revision:* 5 ~ 

### getBond

Returns the bond status of a given `address`.

```python
def getBond(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description      |
|:--------|:--------|:-----------------|
| address | Address | address to query |

*Returns:*

| Key         | Value Type                | Description                                                                  |
|:------------|:--------------------------|:-----------------------------------------------------------------------------|
| totalBonded | int                       | The sum of bond amount                                                       |
| votingPower | int                       | Remaining amount of stake that ICONist can delegate and bond to other P-Reps |
| bonds       | List\[[Vote](#vote)\]     | List of bond information (MAX: 100 entries)                                  |
| unbonds     | List\[[Unbond](#unbond)\] | List of unbond information (MAX: 100 entries)                                |

*Revision:* 13 ~ 

### queryIScore

Returns the amount of I-Score that an `address` has received as a reward.

```python
def queryIScore(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description      |
|:--------|:--------|:-----------------|
| address | Address | address to query |

*Returns:*

| Key          | Value Type | Description                                      |
|:-------------|:-----------|:-------------------------------------------------|
| blockHeight  | int        | block height when I-Score is estimated           |
| iscore       | int        | amount of I-Score                                |
| estimatedICX | int        | estimated amount in loop. 1000 I-Score == 1 loop |

*Revision:* 5 ~

### getPRep

Returns P-Rep register information of a given `address`.

```python
def getPRep(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description      |
|:--------|:--------|:-----------------|
| address | Address | address to query |

*Returns:*

* [PRep](#prep)

*Revision:* 5 ~

### getPReps

Returns the status of all registered P-Rep candidates in descending order by power amount.

```python
def getPReps(startRanking: int, endRanking: int) -> dict:
```

*Parameters:*

| Name         | Type    | Description                                                          |
|:-------------|:--------|:---------------------------------------------------------------------|
| startRanking | int     | (Optional) default: 1<br/>P-Rep list which starts from start ranking |
| endRanking   | int     | (Optional) default: the last ranking                                 |

*Returns:*

| Key            | Value Type            | Description                                         |
|:---------------|:----------------------|:----------------------------------------------------|
| blockHeight    | int                   | latest block height when this request was processed |
| preps          | List\[[PRep](#prep)\] | P-Rep list                                          |
| startRanking   | int                   | start ranking of P-Rep list                         |
| totalDelegated | int                   | total delegation amount that all P-Reps receive     |
| totalStake     | int                   | sum of ICX that all ICONist stake                   |

*Revision:* 5 ~

### getBonderList

Returns the allowed bonder list of a given `address`.

```python
def getBonderList(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description      |
|:--------|:--------|:-----------------|
| address | Address | address to query |

*Returns:*

| Key        | Value Type  | Description                                        |
|:-----------|:------------|:---------------------------------------------------|
| bonderList | \[\]Address | addresses of ICONist who can bond to the `address` |

*Revision:* 13 ~

## Writable APIs

### setStake

Stake some amount of ICX.

```python
def setStake(value: int) -> None:
```

*Parameters:*

| Name  | Type | Description             |
|:------|:-----|:------------------------|
| value | int  | amount of stake in loop |

*Revision:* 5 ~

### setDelegation

Delegate some amount of stake to P-Reps.

- Maximum number of P-Reps to delegate is 100
- The transaction which has duplicated P-Rep addresses will be failed
- This transaction overwrites the previous delegate information

```python
def setDelegation(delegations: List[Vote]) -> None:
```

*Parameters:*

| Name        | Type                  | Description                    |
|:------------|:----------------------|:-------------------------------|
| delegations | List\[[Vote](#vote)\] | list of delegation information |

*Revision:* 5 ~

### setBond

Bond some amount of stake to P-Reps.

- Maximum number of P-Reps to bond is 100
- The transaction which has duplicated P-Rep addresses will be failed
- This transaction overwrites the previous bond information

```python
def setBond(bonds: List[Vote]) -> None:
```

*Parameters:*

| Name  | Type                  | Description              |
|:------|:----------------------|:-------------------------|
| bonds | List\[[Vote](#vote)\] | list of bond information |

*Revision:* 5 ~

### claimIScore

Claim the total reward that a ICONist has received.

```python
def claimIScore() -> None:
```

*Event Log:*

```python
@eventlog(indexed=1)
def IScoreClaimedV2(address: Address, iscore: int, icx: int) -> None:
```
| Name    | Type    | Description                                |
|:--------|:--------|:-------------------------------------------|
| address | Address | address of the ICONist who claimed I-Score |
| iscore  | int     | amount of claimed I-Score                  |
| icx     | int     | amount of claimed I-Score in loop          |

*Revision:* 5 ~

### registerPRep

Register an ICONist as a P-Rep.

- 2000 ICX are required as a registration fee

```python
def registerPRep(name: str, email: str, website: str, country: str, city: str, details: str, p2pEndpoint: str, nodeAddress: Address) -> None:
```

*Parameters:*

| Name        | Type | Description                                                                                                                                                                                               |
|:------------|:-----|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| name        | str  | P-Rep name                                                                                                                                                                                                |
| email       | str  | P-Rep email                                                                                                                                                                                               | 
| website     | str  | P-Rep homepage URL                                                                                                                                                                                        |
| country     | str  | [ISO 3166-1 ALPHA-3](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3)                                                                                                                                    |
| city        | str  | "Seoul", "New York", "Paris"                                                                                                                                                                              |
| details     | str  | URL including P-Rep detail information. See [JSON Standard for P-Rep Detailed Information](https://docs.icon.community/v/icon1/references/reference-manuals/json-standard-for-p-rep-detailed-information) |
| p2pEndpoint | str  | network info used for connecting among P-Rep nodes<br/>"123.45.67.89:7100", "node.example.com:7100"                                                                                                       |
| nodeAddress | str  | (Optional) address of the node key if it's different from the address of P-Rep                                                                                                                            |

*Event Log:*

```python
@eventlog(indexed=0)
def PRepRegistered(address: Address) -> None:
```
| Name    | Type    | Description                 |
|:--------|:--------|:----------------------------|
| address | Address | address of registered P-Rep |

*Revision:* 5 ~

### setPRep

Update P-Rep register information.

```python
def setPRep(name: str, email: str, website: str, country: str, city: str, details: str, p2pEndpoint: str, nodeAddress: Address) -> None:
```

*Parameters:*

| Name        | Type | Description                                                                                                                                                                                                          |
|:------------|:-----|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| name        | str  | (Optional) P-Rep name                                                                                                                                                                                                |
| email       | str  | (Optional) P-Rep email                                                                                                                                                                                               | 
| website     | str  | (Optional) P-Rep homepage URL                                                                                                                                                                                        |
| country     | str  | (Optional) [ISO 3166-1 ALPHA-3](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3)                                                                                                                                    |
| city        | str  | (Optional) "Seoul", "New York", "Paris"                                                                                                                                                                              |
| details     | str  | (Optional) URL including P-Rep detail information. See [JSON Standard for P-Rep Detailed Information](https://docs.icon.community/v/icon1/references/reference-manuals/json-standard-for-p-rep-detailed-information) |
| p2pEndpoint | str  | (Optional) network information used for connecting among P-Rep nodes<br/>Example: "123.45.67.89:7100", "node.example.com:7100"                                                                                       |
| nodeAddress | str  | (Optional) address of the node key if it's different from the address of P-Rep                                                                                                                                       |

*Event Log:*

```python
@eventlog(indexed=0)
def PRepSet(address: Address) -> None:
```
| Name    | Type    | Description                   |
|:--------|:--------|:------------------------------|
| address | Address | address of the modified P-Rep |

*Revision:* 5 ~

### unregisterPRep

Unregister a P-Rep.

```python
def unregisterPRep() -> None:
```

*Event Log:*

```python
@eventlog(indexed=0)
def PRepUnregistered(address: Address) -> None:
```
| Name    | Type    | Description                       |
|:--------|:--------|:----------------------------------|
| address | Address | address of the unregistered P-Rep |

*Revision:* 5 ~

### setBonderList

Set allowed bonder list of P-Rep.

- Maximum number of allowed ICONist to bond is 10
- This transaction overwrites the previous bonder list information

```python
def setBonderList(bonderList: List[Address]) -> None:
```
*Parameters:*

| Name       | Type            | Description                                    |
|:-----------|:----------------|:-----------------------------------------------|
| bonderList | List\[Address\] | addresses of ICONist who can bond to the P-Rep |

*Revision:* 13 ~

# BTP

## ReadOnly APIs

### getBTPNetworkTypeID

Returns BTP Network Type ID of a given `name`.

```python
def getBTPNetworkTypeID(name: str) -> int:
```

*Parameters:*

| Name | Type | Description                  |
|:-----|:-----|:-----------------------------|
| name | str  | name of the BTP Network Type |

*Returns:*

* an int value greater than 0 if BTP Network Type is active.
* an int value 0 if BTP Network Type is not active.
* an int value -1 if BTP Network Type is not supported.

*Revision:* 21 ~

### getPRepNodePublicKey

Returns a public key for the P-Rep node address.

```python
def getPRepNodePublicKey(address: Address) -> bytes:
```

*Parameters:*

| Name    | Type    | Description          |
|:--------|:--------|:---------------------|
| address | Address | address of the P-Rep |

*Returns:*

* the public key or `null` if the P-Rep does not have a public key

*Revision:* 21 ~

## Writable APIs

### openBTPNetwork

Open a BTP Network.

```python
def openBTPNetwork(networkTypeName: str, name: str, owner: Address) -> int:
```

*Parameters:*

| Name            | Type    | Description                                                    |
|:----------------|:--------|:---------------------------------------------------------------|
| networkTypeName | str     | name of the BTP Network Type                                   |
| name            | str     | name of the BTP Network                                        |
| owner           | Address | owner of the BTP Network. Only the owner can send BTP messages |

*Returns:*

* BTP Network ID or 0 if opening a BTP Network is failed

*Event Log:*

```python
@eventlog(indexed=2)
def BTPNetworkTypeActivated(networkTypeName: str, networkTypeId: int) -> None:
```
| Name            | Type | Description                            |
|:----------------|:-----|:---------------------------------------|
| networkTypeName | str  | name of the activated BTP Network Type |
| networkTypeId   | int  | ID of the activated BTP Network Type   |


```python
@eventlog(indexed=2)
def BTPNetworkOpened(networkTypeId: int, networkId: int) -> None:
```
| Name          | Type | Description                  |
|:--------------|:-----|:-----------------------------|
| networkTypeId | int  | ID of the BTP Network Type   |
| networkId     | int  | ID of the opened BTP Network |

*Revision:* 21 ~

### closeBTPNetwork

Close a BTP Network.

```python
def closeBTPNetwork(id: int) -> None:
```

*Parameters:*

| Name | Type | Description    |
|:-----|:-----|:---------------|
| id   | int  | BTP Network ID |

*Event Log:*

```python
@eventlog(indexed=2)
def BTPNetworkClosed(networkTypeId: int, networkId: int) -> None:
```
| Name          | Type | Description                  |
|:--------------|:-----|:-----------------------------|
| networkTypeId | int  | ID of the BTP Network Type   |
| networkId     | int  | ID of the closed BTP Network |

*Revision:* 21 ~

### sendBTPMessage

Send a BTP message over the BTP Network. Only the owner of a BTP Network can send a BTP message.

```python
def sendBTPMessage(networkId: int, message: bytes) -> None:
```

*Parameters:*

| Name      | Type  | Description    |
|:----------|:------|:---------------|
| networkId | str   | BTP Network ID |
| message   | bytes | BTP message    |

*Event Log:*

```python
@eventlog(indexed=2)
def BTPMessage(networkId: int, messageSN: int) -> None:
```
| Name      | Type | Description                            |
|:----------|:-----|:---------------------------------------|
| networkId | int  | ID of the BTP Network                  |
| messageSN | int  | message sequence number in BTP Network |

*Revision:* 21 ~

### registerPRepNodePublicKey

Register an initial public key for the P-Rep node address.

```python
def registerPRepNodePublicKey(address: Address, pubKey: bytes) -> None:
```

*Parameters:*

| Name    | Type    | Description      |
|:--------|:--------|:-----------------|
| address | Address | address of P-Rep |
| pubKey  | bytes   | public key       |

*Revision:* 21 ~

### setPRepNodePublicKey

Set a public key for the P-Rep node address.

```python
def setPRepNodePublicKey(pubKey: bytes) -> None:
```

*Parameters:*

| Name   | Type  | Description |
|:-------|:------|:------------|
| pubKey | bytes | public key  |

*Revision:* 21 ~

# Types

## Unstake

| Key                | Value Type | Description                              |
|:-------------------|:-----------|:-----------------------------------------|
| unstake            | int        | amount of unstake in loop                |
| unstakeBlockHeight | int        | block height when unstake will be done   |
| remainingBlocks    | int        | remaining blocks to `unstakeBlockHeight` |

## Vote

| Key     | Value Type | Description              |
|:--------|:-----------|:-------------------------|
| address | Address    | address of P-Rep to vote |
| value   | int        | vote amount in loop      |

## Unbond

| Key               | Value Type | Description                           |
|:------------------|:-----------|:--------------------------------------|
| address           | Address    | address of P-Rep to bond              |
| value             | int        | bond amount in loop                   |
| expireBlockHeight | int        | block height when unbond will be done |

## PRep

| Key                    | Value Type | Description                                                                                                                                                                                               |
|:-----------------------|:-----------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| address                | Address    | P-Rep address                                                                                                                                                                                             |
| bonded                 | int        | bond amount that a P-Rep receives from ICONist                                                                                                                                                            |
| city                   | str        | "Seoul", "New York", "Paris"                                                                                                                                                                              |
| country                | str        | [ISO 3166-1 ALPHA-3](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3)                                                                                                                                    |
| delegated              | int        | delegation amount that a P-Rep receives from ICONist                                                                                                                                                      |
| details                | str        | URL including P-Rep detail information. See [JSON Standard for P-Rep Detailed Information](https://docs.icon.community/v/icon1/references/reference-manuals/json-standard-for-p-rep-detailed-information) |
| email                  | str        | P-Rep email                                                                                                                                                                                               |
| grade                  | int        | 0: Main P-Rep, 1: Sub P-Rep, 2: P-Rep candidate                                                                                                                                                           |
| hasPubKey              | bool       | (Optional) P-Rep has valid public keys for all active BTP Network type                                                                                                                                    |
| irep                   | int        | incentive rep used to calculate the reward for P-Rep<br>Limit: +- 20% of the previous value                                                                                                               |
| irepUpdatedBlockHeight | int        | block height when a P-Rep changed I-Rep value                                                                                                                                                             |
| lastHeight             | int        | latest block height at which the P-Rep's voting status changed                                                                                                                                            |
| name                   | str        | P-Rep name                                                                                                                                                                                                |
| nodeAddress            | str        | node Key for only consensus                                                                                                                                                                               |
| p2pEndpoint            | str        | network information used for connecting among P-Rep nodes                                                                                                                                                 |
| penalty                | int        | 0: None, 1: Disqualification, 2: Low Productivity, 3: Block Validation, 4: NonVote                                                                                                                        |
| power                  | int        | amount of power that a P-Rep receives from ICONist. (= min(`bonded`+`delegated`, `bonded` * 20))                                                                                                             |
| status                 | int        | 0: active, 1: unregistered                                                                                                                                                                                |
| totalBlocks            | int        | number of blocks that a P-Rep received when running as a Main P-Rep                                                                                                                                       |
| validatedBlocks        | int        | number of blocks that a P-Rep validated when running as a Main P-Rep                                                                                                                                      |
| website                | str        | P-Rep homepage URL                                                                                                                                                                                        |
