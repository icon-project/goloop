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
  * [T_UNSTAKE](#t-unstake)
  * [T_VOTE](#t-vote)
  * [T_UNBOND](#t-unbond)
  * [T_PREP](#t-prep)

# IISS

## ReadOnly APIs

### getStake

Returns the stake status of a given `address`.

```python
def getStake(address: Address) -> dict:
```

*Parameters:*

| Name    | Type      | Description           |
|:--------|:----------|:----------------------|
| address | Address   | the address to query  |

*Returns:*

| Name     | Type                        | Description                 |
|:---------|:----------------------------|:----------------------------|
| stake    | int                         | ICX amount of stake in loop |
| unstakes | \[\][T_UNSTAKE](#t_unstake) | List of Unstake information |

### getDelegation

Returns the delegation status of a given `address`.

```python
def getDelegation(address: Address) -> dict:
```

*Parameters:*

| Name    | Type      | Description           |
|:--------|:----------|:----------------------|
| address | Address   | the address to query  |

*Returns:*

| Key            | Value type            | Description                                                                  |
|:---------------|:----------------------|:-----------------------------------------------------------------------------|
| totalDelegated | int                   | The sum of delegation amount                                                 |
| votingPower    | int                   | Remaining amount of stake that ICONist can delegate and bond to other P-Reps |
| delegations    | \[\][T_VOTE](#t_vote) | List of delegation information (MAX: 100 entries)                            |

### getBond

Returns the bond status of a given `address`.

```python
def getBond(address: Address) -> dict:
```

*Parameters:*

| Name    | Type      | Description           |
|:--------|:----------|:----------------------|
| address | Address   | the address to query  |

*Returns:*

| Name        | Type                      | Description                                                                  |
|:------------|:--------------------------|:-----------------------------------------------------------------------------|
| totalBonded | int                       | The sum of bond amount                                                       |
| votingPower | int                       | Remaining amount of stake that ICONist can delegate and bond to other P-Reps |
| bonds       | \[\][T_VOTE](#t_vote)     | List of bond information (MAX: 100 entries)                                  |
| unbonds     | \[\][T_UNBOND](#t_unbond) | List of unbond information (MAX: 100 entries)                                |

### queryIScore

Returns the amount of I-Score that an `address` has received as a reward.

```python
def queryIScore(address: Address) -> dict:
```

*Parameters:*

| Name    | Type      | Description           |
|:--------|:----------|:----------------------|
| address | Address   | the address to query  |

*Returns:*

| Name         | Type | Description                                      |
|:-------------|:-----|:-------------------------------------------------|
| blockHeight  | int  | Block height when I-Score is estimated           |
| iscore       | int  | Amount of I-Score                                |
| estimatedICX | int  | Estimated amount in loop. 1000 I-Score == 1 loop |

### getPRep

Returns P-Rep register information of a given `address`.

```python
def getPRep(address: Address) -> dict:
```

*Parameters:*

| Name    | Type      | Description           |
|:--------|:----------|:----------------------|
| address | Address   | the address to query  |

*Returns:*

* [T_PREP](#t_prep)

### getPReps

Returns the status of all registered P-Rep candidates in descending order by power amount.

```python
def getPReps(address: Address) -> dict:
```

*Parameters:*

| Name         | Type | Description                                               |
|:-------------|:-----|:----------------------------------------------------------|
| startRanking | int  | Default: 1<br/>P-Rep list which starts from start ranking |
| endRanking   | int  | Default: the last ranking                                 |

*Returns:*

| Name           | Type                  | Description                                             |
|:---------------|:----------------------|:--------------------------------------------------------|
| blockHeight    | int                   | The latest block height when this request was processed |
| preps          | \[\][T_PREP](#t_prep) | P-Rep list                                              |
| startRanking   | int                   | Start ranking of P-Rep list                             |
| totalDelegated | int                   | Total delegation amount that all P-Reps receive         |
| totalStake     | int                   | The sum of ICX that all ICONist stake                   |

### getBonderList

Returns the allowed bonder list of a given `address`.

```python
def getBonderList(address: Address) -> dict:
```

*Parameters:*

| Name    | Type      | Description           |
|:--------|:----------|:----------------------|
| address | Address   | the address to query  |

*Returns:*

| Name       | Type        | Description      |
|:-----------|:------------|:-----------------|
| bonderList | \[\]Address | List of address  |

## Writable APIs

### setStake

Stake some amount of ICX.

```python
def setStake(value: int) -> None:
```

*Parameters:*

| Name  | Type  | Description                     |
|:------|:------|:--------------------------------|
| value | int   | the ICX amount of stake in loop |

### setDelegation

Delegate some amount of stake to P-Reps.

- Maximum number of P-Reps to delegate is 100
- The transaction which has duplicated P-Rep addresses will be failed
- This transaction overwrites the previous delegate information

```python
def setDelegation(delegations: list) -> None:
```

*Parameters:*

| Name        | Type                  | Description                    |
|:------------|:----------------------|:-------------------------------|
| delegations | \[\][T_VOTE](#t_vote) | List of delegation information |

### setBond

Bond some amount of stake to P-Reps.

- Maximum number of P-Reps to bond is 100
- The transaction which has duplicated P-Rep addresses will be failed
- This transaction overwrites the previous bond information

```python
def setBond(bonds: list) -> None:
```

*Parameters:*

| Name  | Type                  | Description              |
|:------|:----------------------|:-------------------------|
| bonds | \[\][T_VOTE](#t_vote) | List of bond information |

### claimIScore

Claim the total reward that a ICONist has received.

```python
def claimIScore() -> None:
```

*Event Log:*

```python
@EventLog(indexed=1)
def IScoreClaimedV2(address: Address, iscore: int, icx: int) -> None: pass
```

### registerPRep

Register an ICONist as a P-Rep.

- 2000 ICX are required as a registration fee

```python
def registerPRep(name: string, email: string, website: string, country: string, city: string, details: string,
                 p2pEndpoint: string, nodeAddress: Address) -> None:
```

*Parameters:*

| Name        | Type    | Description                                                                                         |
|:------------|:--------|:----------------------------------------------------------------------------------------------------|
| name        | str     | P-Rep name                                                                                          |
| email       | str     | P-Rep email                                                                                         | 
| website     | str     | P-Rep homepage url                                                                                  |
| country     | str     | [ISO 3166-1 ALPHA-3](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3)                              |
| city        | str     | "Seoul", "New York", "Paris"                                                                        |
| details     | str     | Url including P-Rep detail information                                                              |
| p2pEndpoint | str     | Network info used for connecting among P-Rep nodes<br/>"123.45.67.89:7100", "node.example.com:7100" |
| nodeAddress | str     | (Optional) Node Key for only consensus                                                              |

*details :
See [JSON Standard for P-Rep Detailed Information](https://www.icondev.io/docs/json-standard-for-p-rep-detailed-information)

*Event Log:*

```python
@EventLog(indexed=0)
def PRepRegistered(address: Address) -> None: pass
```

### setPRep

Update P-Rep register information.

```python
def setPRep(name: string, email: string, website: string, country: string, city: string, details: string,
            p2pEndpoint: string, nodeAddress: Address) -> None:
```

*Parameters:*

| Name        | Type  | Description                                                                                                    |
|:------------|:------|:---------------------------------------------------------------------------------------------------------------|
| name        | str   | (Optional) P-Rep name                                                                                          |
| email       | str   | (Optional) P-Rep email                                                                                         | 
| website     | str   | (Optional) P-Rep homepage url                                                                                  |
| country     | str   | (Optional) [ISO 3166-1 ALPHA-3](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3)                              |
| city        | str   | (Optional) "Seoul", "New York", "Paris"                                                                        |
| details     | str   | (Optional) Url including P-Rep detail information                                                              |
| p2pEndpoint | str   | (Optional) Network info used for connecting among P-Rep nodes<br/>"123.45.67.89:7100", "node.example.com:7100" |
| nodeAddress | str   | (Optional) Node Key for only consensus                                                                         |

*details :
See [JSON Standard for P-Rep Detailed Information](https://www.icondev.io/docs/json-standard-for-p-rep-detailed-information)

*Event Log:*

```python
@EventLog(indexed=0)
def PRepSet(address: Address) -> None: pass
```

### unregisterPRep

Unregister a P-Rep.

```python
def unregisterPRep() -> None:
```

*Event Log:*

```python
@EventLog(indexed=0)
def PRepUnregistered(address: Address) -> None: pass
```

### setBonderList

Set allowed bonder list of P-Rep.

- Maximum number of allowed ICONist to bond is 10
- This transaction overwrites the previous bonder list information

```python
def setBonderList(bonderList: list) -> None:
```
*Parameters:*

| Name       | Type        | Description                       |
|:-----------|:------------|:----------------------------------|
| bonderList | \[\]Address | List of address (MAX: 10 entries) |

# BTP

## ReadOnly APIs

### getBTPNetworkTypeID

Returns BTP Network Type ID of a given `name`.

```python
def getBTPNetworkTypeID(name: str) -> int:
```

*Parameters:*

| Name | Type  | Description                   |
|:-----|:------|:------------------------------|
| name | str   | the name of BTP Network Type  |

*Returns:*

* an int value greater than 0 if BTP Network Type is active.
* an int value 0 if BTP Network Type is not active.
* an int value -1 if BTP Network Type is not supported.

### getPRepNodePublicKey

Returns a public key for the P-Rep node address.

```python
def getPRepNodePublicKey(address: Address) -> bytes:
```

*Parameters:*

| Name    | Type    | Description          |
|:--------|:--------|:---------------------|
| address | Address | the address of P-Rep |

*Returns:*

* the public key or 'null' if the P-Rep does not have a public key

## Writable APIs

### openBTPNetwork

Open a BTP Network.

```python
def openBTPNetwork(networkTypeName: str, name: str, owner: Address) -> int:
```

*Parameters:*

| Name            | Type    | Description                  |
|:----------------|:--------|:-----------------------------|
| networkTypeName | str     | the name of BTP Network Type |
| name            | str     | the name of BTP Network      |
| owner           | Address | the owner of BTP Network     |

*Returns:*

* BTP Network ID or 0 if opening a BTP Network is failed

*Event Log:*

```python
@EventLog(indexed=2)
def BTPNetworkTypeActivated(networkTypeName: str, networkTypeId: int) -> None: pass


@EventLog(indexed=2)
def BTPNetworkOpened(networkTypeId: int, networkId: int) -> None: pass
```

### closeBTPNetwork

Close a BTP Network.

```python
def closeBTPNetwork(id: int) -> None:
```

*Parameters:*

| Name | Type | Description        |
|:-----|:-----|:-------------------|
| id   | int  | the BTP Network ID |

*Event Log:*

```python
@EventLog(indexed=2)
def BTPNetworkClosed(networkTypeId: int, networkId: int) -> None: pass
```

### sendBTPMessage

Send a BTP message over the BTP Network. Only the owner of a BTP Network can send a BTP message.

```python
def sendBTPMessage(networkId: int, message: bytes) -> None:
```

*Parameters:*

| Name      | Type  | Description        |
|:----------|:------|:-------------------|
| networkId | str   | the BTP Network ID |
| message   | bytes | BTP message        |

*Event Log:*

```python
@EventLog(indexed=2)
def BTPMessage(networkId: int, messageSN: int) -> None: pass
```

### registerPRepNodePublicKey

Register an initial public key for the P-Rep node address.

```python
def registerPRepNodePublicKey(address: Address, pubKey: bytes) -> None:
```

*Parameters:*

| Name    | Type    | Description          |
|:--------|:--------|:---------------------|
| address | Address | the address of P-Rep |
| pubKey  | bytes   | the public key       |

### setPRepNodePublicKey

Set a public key for the P-Rep node address.

```python
def setPRepNodePublicKey(pubKey: bytes) -> None:
```

*Parameters:*

| Name   | Type  | Description    |
|:-------|:------|:---------------|
| pubKey | bytes | the public key |

# Types

## T_UNSTAKE

| Name               | Type | Description                              |
|:-------------------|:-----|:-----------------------------------------|
| unstake            | int  | ICX amount of unstake in loop            |
| unstakeBlockHeight | int  | BlockHeight when unstake will be done    |
| remainingBlocks    | int  | Remaining blocks to `unstakeBlockHeight` |

## T_VOTE

| Name    | Type    | Description              |
|:--------|:--------|:-------------------------|
| address | Address | Address of P-Rep to vote |
| value   | int     | Vote amount in loop      |

## T_UNBOND

| Name               | Type    | Description                          |
|:-------------------|:--------|:-------------------------------------|
| address            | Address | Address of P-Rep to bond             |
| value              | int     | Bond amount in loop                  |
| expireBlockHeight  | int     | BlockHeight when unbond will be done |

## T_PREP

| Name                   | Type    | Description                                                                                 |
|:-----------------------|:--------|:--------------------------------------------------------------------------------------------|
| address                | Address | P-Rep address                                                                               |
| bonded                 | int     | Bond amount that a P-Rep receives from ICONist                                              |
| city                   | str     | "Seoul", "New York", "Paris"                                                                |
| country                | str     | [ISO 3166-1 ALPHA-3](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3)                      |
| delegated              | int     | Delegation amount that a P-Rep receives from ICONist                                        |
| details                | str     | Url including P-Rep detail informatio                                                       |
| email                  | str     | P-Rep email                                                                                 |
| grade                  | int     | 0: Main P-Rep, 1: Sub P-Rep, 2: P-Rep candidate                                             |
| hasPubKey              | bool    | (Optional) P-Rep has valid public keys for all active BTP Network type                      |
| irep                   | int     | Incentive rep used to calculate the reward for P-Rep<br>Limit: +- 20% of the previous value |
| irepUpdatedBlockHeight | int     | Block height when a P-Rep changed I-Rep value                                               |
| lastHeight             | int     | The latest block height at which the P-Rep's voting status changed                          |
| name                   | str     | P-Rep name                                                                                  |
| nodeAddress            | str     | Node Key for only consensus                                                                 |
| p2pEndpoint            | str     | Network info used for connecting among P-Rep nodes                                          |
| penalty                | int     | 0: None, 1: Disqualification, 2: Low Productivity, 3: Block Validation, 4: NonVote          |
| power                  | int     | Amount power that a P-Rep receives from ICONist. (= max(`bonded`+`delegated`, bonded * 20)  |
| status                 | int     | 0: active, 1: unregistered                                                                  |
| totalBlocks            | int     | The number of blocks that a P-Rep received when running as a Main P-Rep                     |
| validatedBlocks        | int     | The number of blocks that a P-Rep validated when running as a Main P-Rep                    |
| website                | str     | P-Rep homepage url                                                                          |
