# ICON Chain SCORE API

<details>
  <summary>Contents</summary>

- APIs
    - [Basic](#basic)
        * ReadOnly APIs
            + [getRevision](#getrevision)
            + [getStepPrice](#getstepprice)
            + [getStepCost](#getstepcost)
            + [getStepCosts](#getstepcosts)
            + [getMaxStepLimit](#getmaxsteplimit)
            + [getScoreStatus](#getscorestatus)
            + [getBlockedScores](#getblockedscores)
            + [getScoreOwner](#getscoreowner)
            + [isBlocked](#isblocked)
        * Writable APIs
            + [setRevision](#setrevision)
            + [setStepPrice](#setstepprice)
            + [setStepCost](#setstepcost)
            + [setMaxStepLimit](#setmaxsteplimit)
            + [disableScore](#disablescore)
            + [enableScore](#enablescore)
            + [blockScore](#blockscore)
            + [unblockScore](#unblockscore)
            + [burn](#burn)
            + [setScoreOwner](#setscoreowner)
            + [blockAccount](#blockaccount)
            + [unblockAccount](#unblockaccount)
     - [IISS](#iiss)
        * ReadOnly APIs
            + [getStake](#getstake)
            + [getDelegation](#getdelegation)
            + [getBond](#getbond)
            + [queryIScore](#queryiscore)
            + [getPRep](#getprep)
            + [getPReps](#getpreps)
            + [getMainPReps](#getmainpreps)
            + [getSubPReps](#getsubpreps)
            + [estimateUnstakeLockPeriod](#estimateunstakelockperiod)
            + [getPRepTerm](#getprepterm)
            + [getBonderList](#getbonderlist)
            + [getPRepStats](#getprepstats)
            + [getNetworkInfo](#getnetworkinfo)
            + [getNetworkScores](#getnetworkscores)
            + [getPRepStatsOf](#getprepstatsof)
            + [getSlashingRates](#getslashingrates)
            + [getMinimumBond](#getminimumbond)
        * Writable APIs
            + [setStake](#setstake)
            + [setDelegation](#setdelegation)
            + [setBond](#setbond)
            + [claimIScore](#claimiscore)
            + [registerPRep](#registerprep)
            + [setPRep](#setprep)
            + [unregisterPRep](#unregisterprep)
            + [disqualifyPRep](#disqualifyprep)
            + [setBonderList](#setbonderlist)
            + [setRewardFund](#setrewardfund)
            + [setRewardFundAllocation](#setrewardfundallocation)
            + [penalizeNonvoters](#penalizenonvoters)
            + [setNetworkScore](#setnetworkscore)
            + [setRewardFundAllocation2](#setrewardfundallocation2)
            + [setMinimumBond](#setminimumbond)
            + [initCommissionRate](#initcommissionrate)
            + [setCommissionRate](#setcommissionrate)
            + [setSlashingRates](#setslashingrates)
            + [requestUnjail](#requestunjail)
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
    * [Value Types](#value-types)
    * [StepCosts](#stepcosts)
    * [Unstake](#unstake)
    * [Vote](#vote)
    * [Unbond](#unbond)
    * [PRep](#prep)
    * [PRepSnapshot](#prepsnapshot)
    * [ContractStatus](#contractstatus)
    * [DepositInfo](#depositinfo)
    * [Deposit](#deposit)
    * [RewardFund](#rewardfund)
    * [NamedValue](#namedvalue)
- [Event logs](#event-logs)
    * [PenaltyImposed(Address,int,int)](#penaltyimposedaddressintint)
- [Predefined variables](#predefined-variables)
    * [PENALTY_TYPE](#penaltytype)
    * [NETWORK_SCORE_TYPE](#networkscoretype)
    * [REWARD_FUND_ALLOCATION_KEY](#rewardfundallocationkey)

</details>

# Basic

## ReadOnly APIs

### getRevision

Returns the revision of the network.

```
def getRevision() -> int:
```

*Returns:*

* the revision of the network

*Revision:* 0 ~

### getStepPrice

Returns the price of step in loop.

```
def getStepPrice() -> int:
```

*Returns:*

* the price of step in loop

*Revision:* 0 ~

### getStepCost

Returns the step cost of given step type `type`.

```
def getStepCost(type: str) -> int:
```

*Parameters:*

| Name | Type | Description                                          |
|:-----|:-----|:-----------------------------------------------------|
| type | str  | step type. refer to `Key` of [StepCosts](#stepcosts) |

*Returns:*

* the step cost of step type

*Revision:* 0 ~

### getStepCosts

Returns the step costs of all step types.

```
def getStepCosts() -> dict:
```

*Returns:*

* [StepCosts](#stepcosts)

*Revision:* 0 ~

### getMaxStepLimit

Returns the maximum value of step limit for the given `contextType`.

```
def getMaxStepLimit(contextType: str) -> int:
```

*Parameters:*

| Name        | Type | Description                   |
|:------------|:-----|:------------------------------|
| contextType | str  | context type. (invoke, query) |

*Returns:*

* the maximum value of step limit

*Revision:* 0 ~

### getScoreStatus

Returns the status of the SCORE.

```
def getScoreStatus(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description               |
|:--------|:--------|:--------------------------|
| address | Address | address of SCORE to query |

*Returns:*

| Key         | Value Type                        | Description                                   |
|:------------|:----------------------------------|:----------------------------------------------|
| owner       | Address                           | owner of the SCORE                            |
| blocked     | [T_BOOL](#T_BOOL)                 | `0x1` if it's blocked by governance           |
| disabled    | [T_BOOL](#T_BOOL)                 | `0x1` if it's disabled by owner               |
| current     | [ContractStatus](#contractstatus) | Current status                                |
| next        | [ContractStatus](#contractstatus) | (Optional) status of next SCORE to be audited |
| depositInfo | [DepositInfo](#depositinfo)       | (Optional) deposit information                |

*Revision:* 0 ~

### getBlockedScores

Returns addresses of blocked SCOREs.

```
def getBlockedScores() -> List[Address]:
```

*Returns:*

* addresses of blocked SCOREs

*Revision:* 9 ~

### getScoreOwner

Returns the owner of the SCORE.

```
def getScoreOwner(address: Address) -> Address:
```

*Parameters:*

| Name    | Type    | Description               |
|:--------|:--------|:--------------------------|
| address | Address | address of SCORE to query |

*Returns:*

* the owner address

*Revision:* 17 ~

### isBlocked

Returns whether it's blocked or not

```
def isBlocked(address: Address) -> bool:
```

*Parameters:*

| Name    | Type    | Description                             |
|:--------|:--------|:----------------------------------------|
| address | Address | address of the account or the contract  |

*Returns:* `True` if it's blocked.

*Revision:* 22 ~

## Writable APIs

### setRevision

Updates the revision of the network. Governance only.

```
def setRevision(code: int) -> None:
```

*Parameters:*

| Name | Type | Description             |
|:-----|:-----|:------------------------|
| code | int  | revision of the network |

*Event Log:*
- from revision 24
```
@eventlog(indexed=0)
def RevisionSet(code: int) -> None:
```

| Name | Type | Description             |
|:-----|:-----|:------------------------|
| code | int  | revision of the network |

*Revision:* 0 ~

### setStepPrice

Updates the price of step in loop. Governance only.

```
def setStepPrice(price: int) -> None:
```

*Parameters:*

| Name  | Type | Description           |
|:------|:-----|:----------------------|
| price | int  | price of step in loop |

*Event Log:*
- from revision 24
```
@eventlog(indexed=0)
def StepPriceSet(price: int) -> None:
```

| Name  | Type | Description            |
|:------|:-----|:-----------------------|
| price | int  | price of step in loop  |

*Revision:* 0 ~

### setStepCost

Updates the step cost of given `type` step type. Governance only.

```
def setStepCost(type: str, cost: int) -> None:
```

*Parameters:*

| Name | Type | Description                                          |
|:-----|:-----|:-----------------------------------------------------|
| type | str  | step type. refer to `Key` of [StepCosts](#stepcosts) |
| cost | int  | cost for step type                                   |

*Event Log:*
- from revision 24
```
@eventlog(indexed=0)
def StepCostSet(type: str, cost: int) -> None:
```

| Name | Type | Description                                           |
|:-----|:-----|:------------------------------------------------------|
| type | str  | step type. refer to `Key` of [StepCosts](#stepcosts)  |
| cost | int  | cost for step type                                    |

*Revision:* 0 ~

### setMaxStepLimit

Updates the maximum step limit of given `contextType`. Governance only.

```
def setMaxStepLimit(contextType: str, limit: int) -> None:
```

*Parameters:*

| Name        | Type | Description                     |
|:------------|:-----|:--------------------------------|
| contextType | str  | context type. (invoke, query)   |
| limit       | int  | max step limit for context type |

*Event Log:*
- from revision 24
```
@eventlog(indexed=0)
def MaxStepLimitSet(contextType: str, limit: int) -> None:
```

| Name        | Type | Description                     |
|:------------|:-----|:--------------------------------|
| contextType | str  | context type. (invoke, query)   |
| limit       | int  | max step limit for context type |

*Revision:* 0 ~

### disableScore

Disables the SCORE. Allowed only from the SCORE owner.

```
def disableScore(address: Address) -> None:
```

*Parameters:*

| Name    | Type    | Description          |
|:--------|:--------|:---------------------|
| address | Address | address of the SCORE |

*Revision:* 0 ~

### enableScore

Enables the SCORE. Allowed only from the SCORE owner.

```
def enableScore(address: Address) -> None:
```

*Parameters:*

| Name    | Type    | Description          |
|:--------|:--------|:---------------------|
| address | Address | address of the SCORE |

*Revision:* 0 ~

### blockScore

Blocks the SCORE. Governance only.

```
def blockScore(address: Address) -> None:
```

*Parameters:*

| Name    | Type    | Description          |
|:--------|:--------|:---------------------|
| address | Address | address of the SCORE |

*Event Log:*
- from revision 24
```
@eventlog(indexed=1)
def AccountBlockedSet(address: Address, yn: bool) -> None:
```

| Name     | Type    | Description          |
|:---------|:--------|:---------------------|
| address  | Address | address of the SCORE |
| yn       | bool    | blocked or not       |

*Revision:* 0 ~

### unblockScore

Unblocks the SCORE. Governance only.

```
def unblockScore(address: Address) -> None:
```

*Parameters:*

| Name    | Type    | Description          |
|:--------|:--------|:---------------------|
| address | Address | address of the SCORE |

*Event Log:*
- from revision 24
```
@eventlog(indexed=1)
def AccountBlockedSet(address: Address, yn: bool) -> None:
```

| Name     | Type     | Description           |
|:---------|:---------|:----------------------|
| address  | Address  | address of the SCORE  |
| yn       | bool     | blocked or not        |

*Revision:* 0 ~

### burn

Burns the balance of the sender. Set amount with `value` of `icx_sendTransaction`.

```
def burn() -> None:
```

*Event Log:*

```
@eventlog(indexed=1)
def ICXBurnedV2(address: Address, amount: int, total_supply: int) -> None:
```

| Name         | Type    | Description                               |
|:-------------|:--------|:------------------------------------------|
| address      | Address | address of the ICONist who burned balance |
| amount       | int     | amount of burned balance                  |
| total_supply | int     | amount of total supply after burn         |

*Revision:* 12 ~

### setScoreOwner

Updates the owner of the SCORE. Allowed only from the SCORE owner.

- Not allowed for blocked or disabled SCORE.

```
def setScoreOwner(score: Address, owner: Address) -> None:
```

*Parameters:*

| Name  | Type    | Description          |
|:------|:--------|:---------------------|
| score | Address | address of the SCORE |
| owner | Address | address of new owner |

*Revision:* 17 ~

### blockAccount

It blocks the account (EoA). It's only for governance.
If it's already blocked, then it ignores silently.
Otherwise, it emit the event.

```
def blockAccount(address: Address) -> None:
```

*Parameters:*

| Name    | Type    | Description                     |
|:--------|:--------|:--------------------------------|
| address | Address | address of the account to block |

*Event Log:*

```
@eventlog(indexed=1)
def AccountBlockedSet(address: Address, yn: bool) -> None:
```

*Revision:* 22 ~

### unblockAccount

It unblocks the account (EoA). It's only for governance.
If it's already unblocked, then it silently ignores.
Otherwise, it emit the event.

```
def unblockAccount(address: Address) -> None:
```

*Parameters:*

| Name    | Type    | Description                     |
|:--------|:--------|:--------------------------------|
| address | Address | address of the account to block |

*Event Log:*

```
@eventlog(indexed=1)
def AccountBlockedSet(address: Address, yn: bool) -> None:
```

*Revision:* 22 ~

# IISS

## ReadOnly APIs

### getStake

Returns the stake status of the given `address`.

```
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

Returns the delegation status of the given `address`.

```
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

Returns the bond status of the given `address`.

```
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

Returns the amount of I-Score that `address` has received as a reward.

```
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

Returns P-Rep register information of the given `address`.

```
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

```
def getPReps(startRanking: int, endRanking: int) -> dict:
```

*Parameters:*

| Name         | Type | Description                                                          |
|:-------------|:-----|:---------------------------------------------------------------------|
| startRanking | int  | (Optional) default: 1<br/>P-Rep list which starts from start ranking |
| endRanking   | int  | (Optional) default: the last ranking                                 |

*Returns:*

| Key            | Value Type            | Description                                         |
|:---------------|:----------------------|:----------------------------------------------------|
| blockHeight    | int                   | latest block height when this request was processed |
| preps          | List\[[PRep](#prep)\] | P-Rep list                                          |
| startRanking   | int                   | start ranking of P-Rep list                         |
| totalDelegated | int                   | total delegation amount that all P-Reps receive     |
| totalStake     | int                   | sum of ICX that all ICONist stake                   |

*Revision:* 5 ~

### getMainPReps

Returns the list of all Main P-Reps in descending order by power amount.

```
def getMainPReps() -> dict:
```

*Returns:*

| Key            | Value Type                            | Description                                         |
|:---------------|:--------------------------------------|:----------------------------------------------------|
| blockHeight    | int                                   | latest block height when this request was processed |
| preps          | List\[[PRepSnapshot](#prepsnapshot)\] | P-Rep list                                          |
| totalDelegated | int                                   | total delegation amount that Main P-Reps receive    |
| totalPower     | int                                   | total power amount that Main P-Reps receive         |

*Revision:* 5 ~

### getSubPReps

Returns the list of all Sub P-Reps in descending order by power amount.

```
def getSubPReps() -> dict:
```

*Returns:*

| Key            | Value Type                            | Description                                         |
|:---------------|:--------------------------------------|:----------------------------------------------------|
| blockHeight    | int                                   | latest block height when this request was processed |
| preps          | List\[[PRepSnapshot](#prepsnapshot)\] | P-Rep list                                          |
| totalDelegated | int                                   | total delegation amount that Sub P-Reps receive     |
| totalPower     | int                                   | total power amount that Sub P-Reps receive          |

*Revision:* 5 ~

### estimateUnstakeLockPeriod

Returns estimated unstake lock period.

```
def estimateUnstakeLockPeriod() -> dict:
```

*Returns:*

| Key               | Value Type | Description                   |
|:------------------|:-----------|:------------------------------|
| unstakeLockPeriod | int        | estimated unstake lock period |

*Revision:* 5 ~

### getPRepTerm

Returns information for the current term.

```
def getPRepTerm() -> dict:
```

*Returns:*

| Key              | Value Type                            | Description                                                 |
|:-----------------|:--------------------------------------|:------------------------------------------------------------|
| blockHeight      | int                                   | latest block height when this request was processed         |
| sequence         | int                                   | sequence number                                             |
| startBlockHeight | int                                   | start block height of the term                              |
| endBlockHeight   | int                                   | end block height of the term                                |
| totalSupply      | int                                   | total supply amount at `startBlockHeight`                   |
| preps            | List\[[PRepSnapshot](#prepsnapshot)\] | Main/Sub P-Rep list at `startBlockHeight`                   |
| totalDelegated   | int                                   | total delegation amount of `preps`                          |
| totalPower       | int                                   | total power amount of `preps`                               |
| period           | int                                   | term period                                                 |
| rewardFund       | [RewardFund](#rewardfund)             | reward fund information for the term                        |
| bondRequirement  | int                                   | bondRequriement for the term                                |
| revision         | int                                   | revision for the term                                       |
| isDecentralized  | bool                                  | `true` if network is decentralized                          |
| mainPRepCount    | int                                   | Main P-Reps count for the term                              |
| iissVersion      | int                                   | IISS version for the term                                   |
| irep             | int                                   | (Optional. revision < 25) Irep for the term                  |
| rrep             | int                                   | (Optional. revision < 25) Rrep for the term                 |
| minimumBond      | int                                   | (Optional. revision >= 25) minimum bond amount for the term |

*Revision:* 5 ~

### getBonderList

Returns the list of allowed bonders for the given `address`.

```
def getBonderList(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description      |
|:--------|:--------|:-----------------|
| address | Address | address to query |

*Returns:*

| Key        | Value Type    | Description                                        |
|:-----------|:--------------|:---------------------------------------------------|
| bonderList | List[Address] | addresses of ICONist who can bond to the `address` |

*Revision:* 13 ~

### getPRepStats

Returns the list of block validation statistics for all active PReps

```
def getPRepStats() -> dict:
```

*Returns:*

| Key         | Type                            | Description                                             |
|:------------|:--------------------------------|:--------------------------------------------------------|
| blockHeight | int                             | state blockHeight                                       |
| preps       | List\[[PRepStats](#prepstats)\] | List of block validation statistics for all active PRep |

*Revision:* 13 ~

### getNetworkInfo

Returns the configuration and status of the network.

```
def getNetworkInfo() -> dict:
```

*Returns:*

| Key               | Value Type                | Description                                |
|:------------------|:--------------------------|:-------------------------------------------|
| mainPRepCount     | int                       | Main P-Reps count                          |
| extraPRepCount    | int                       | Extra Main P-Reps count                    |
| subPRepCount      | int                       | Sub Main P-Reps count                      |
| iissVersion       | int                       | IISS version                               |
| termPeriod        | int                       | period of term                             |
| bondRequirement   | int                       | bond requirement                           |
| lockMinMultiplier | int                       | multiplier for minimum unstake lock period |
| lockMaxMultiplier | int                       | multiplier for maximum unstake lock period |
| unstakeSlotMax    | int                       | maximum unstakes count of a account        |
| delegationSlotMax | int                       | maximum delegation count of a account      |
| rewardFund        | [RewardFund](#rewardfund) | reward fund information                    |
| totalStake        | int                       | total stakes of ICONist                    |
| totalBonded       | int                       | total bonded amount of P-Rep               |
| totalDelegated    | int                       | total delegated amount of P-Rep            |
| totalPower        | int                       | total power amount of P-Rep                |
| preps             | int                       | count of all P-Reps                        |

*Revision:* 13 ~

### getNetworkScores

Returns the list of network SCOREs

```
def getNetworkScores() -> dict:
```

*Returns:*

| Key                                        | Type    | Description              |
|:-------------------------------------------|:--------|:-------------------------|
| ${[NETWORK_SCORE_TYPE](#networkscoretype)} | Address | address of network SCORE |

*Revision:* 15 ~

### getPRepStatsOf

Returns the list of block validation statistics for the given PRep

```
def getPRepStatsOf(address: Address) -> dict:
```

*Parameters:*

| Name    | Type    | Description                    |
|:--------|:--------|:-------------------------------|
| address | Address | Owner address of PRep to query |

*Returns:*

| Key         | Type                            | Description                                            |
|:------------|:--------------------------------|:-------------------------------------------------------|
| blockHeight | int                             | state blockHeight                                      |
| preps       | List\[[PRepStats](#prepstats)\] | List of block validation statistics for the given PRep |

*Revision:* 22 ~

### getSlashingRates

Returns slashing rates for given `names`

```
def getSlashingRates(names: List[str]) -> dict:
```

*Parameters:*

| Name  | Type        | Description                                                     |
|:------|:------------|:----------------------------------------------------------------|
| names | List\[str\] | list of names to query. Pass an empty list to query all values. |

*Returns:*

| Key                             | Type | Description                          |
|:--------------------------------|:-----|:-------------------------------------|
| ${[PENALTY_TYPE](#penaltytype)} | int  | slashing rate value from 0 to 10,000 |

*Revision:* 24 ~

### getMinimumBond

Returns the minimum bond amount required to earn the minimum wage

```
def getMinimumBond() -> int:
```

*Returns:*

* the minimum bond amount

*Revision:* 24 ~

## Writable APIs

### setStake

Stakes some amount of ICX.

```
def setStake(value: int) -> None:
```

*Parameters:*

| Name  | Type | Description             |
|:------|:-----|:------------------------|
| value | int  | amount of stake in loop |

*Revision:* 5 ~

### setDelegation

Delegates some amount of stake to P-Reps.

- Maximum number of P-Reps to delegate is 100
- The transaction which has duplicated P-Rep addresses will be failed
- This transaction overwrites the previous delegate information

```
def setDelegation(delegations: List[Vote]) -> None:
```

*Parameters:*

| Name        | Type                  | Description                    |
|:------------|:----------------------|:-------------------------------|
| delegations | List\[[Vote](#vote)\] | list of delegation information |

*Event Log:*
- from revision 24
```
@eventlog(indexed=1)
def DelegationSet(address: Address, delegations: bytes) -> None:
```

| Name        | Type    | Description                                 |
|:------------|:--------|:--------------------------------------------|
| address     | Address | address of the delegator                    |
| delegations | bytes   | rlp encoded data of parameter `delegations` |

*Revision:* 5 ~

### setBond

Bonds some amount of stake to P-Reps.

- Maximum number of P-Reps to bond is 100
- The transaction which has duplicated P-Rep addresses will be failed
- This transaction overwrites the previous bond information

```
def setBond(bonds: List[Vote]) -> None:
```

*Parameters:*

| Name  | Type                  | Description              |
|:------|:----------------------|:-------------------------|
| bonds | List\[[Vote](#vote)\] | list of bond information |

*Event Log:*
- from revision 24
```
@eventlog(indexed=1)
def BondSet(address: Address, bonds: bytes) -> None:
```

| Name    | Type    | Description                           |
|:--------|:--------|:--------------------------------------|
| address | Address | address of the bonder                 |
| bonds   | bytes   | rlp encoded data of parameter `bonds` |

*Revision:* 5 ~

### claimIScore

Claims the total reward that a ICONist has received.

```
def claimIScore() -> None:
```

*Event Log:*

```
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

Registers an ICONist as a P-Rep.

- 2000 ICX are required as a registration fee

```
def registerPRep(name: str, email: str, website: str, country: str, city: str, details: str, p2pEndpoint: str,
                 nodeAddress: Address) -> None:
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

```
@eventlog(indexed=0)
def PRepRegistered(address: Address) -> None:
```

| Name    | Type    | Description                 |
|:--------|:--------|:----------------------------|
| address | Address | address of registered P-Rep |

*Revision:* 5 ~

### setPRep

Updates P-Rep's register information.

```
def setPRep(name: str, email: str, website: str, country: str, city: str, details: str, p2pEndpoint: str,
            nodeAddress: Address) -> None:
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

```
@eventlog(indexed=0)
def PRepSet(address: Address) -> None:
```

| Name    | Type    | Description               |
|:--------|:--------|:--------------------------|
| address | Address | address of modified P-Rep |

*Revision:* 5 ~

### unregisterPRep

Unregisters the P-Rep.

```
def unregisterPRep() -> None:
```

*Event Log:*

```
@eventlog(indexed=0)
def PRepUnregistered(address: Address) -> None:
```

| Name    | Type    | Description                   |
|:--------|:--------|:------------------------------|
| address | Address | address of unregistered P-Rep |

*Revision:* 5 ~

### disqualifyPRep

Disqualify the P-Rep. Governance Only.

```
def disqualifyPRep(address: Address) -> None:
```

*Parameters:*

| Name    | Type    | Description                            |
|:--------|:--------|:---------------------------------------|
| address | Address | address of the P-Rep to be disqulified |

*Event Log:*
[PenaltyImposed(Address,int,int)](#penaltyimposedaddressintint)

*Revision:* 6 ~

### setBonderList

Updates allowed bonder list of P-Rep.

- Maximum number of allowed ICONist to bond is 10
- This transaction overwrites the previous bonder list information

```
def setBonderList(bonderList: List[Address]) -> None:
```

*Parameters:*

| Name       | Type            | Description                                    |
|:-----------|:----------------|:-----------------------------------------------|
| bonderList | List\[Address\] | addresses of ICONist who can bond to the P-Rep |

*Revision:* 13 ~

### setRewardFund

Updates the size of the reward fund. Governance only.

```
def setRewardFund(iglobal: int) -> None:
```

*Parameters:*

| Name    | Type | Description         |
|:--------|:-----|:--------------------|
| iglobal | int  | size of reward fund |

*Event Log:*
- from revision 24
```
@eventlog(indexed=0)
def RewardFundSet(iglobal: int) -> None:
```

| Name     | Type | Description          |
|:---------|:-----|:---------------------|
| iglobal  | int  | size of reward fund  |

*Revision:* 13 ~

### setRewardFundAllocation

Updates allocation of reward fund. Governance only.

- Sum of all allocation rates must be 100

```
def setRewardFundAllocation(iprep: int, icps: int, irelay: int, ivoter: int) -> None:
```

*Parameters:*

| Name   | Type | Description                                  |
|:-------|:-----|:---------------------------------------------|
| iprep  | int  | percentage allocated to the P-Rep reward     |
| icps   | int  | percentage allocated to the CPS reward       |
| irelay | int  | percentage allocated to the BTP relay reward |
| ivoter | int  | percentage allocated to the Voter reward     |

*Revision:* 13 ~ 24

### penalizeNonvoters

Penalizes P-Reps for not voting on Network Proposal. Governance Only.

```
def penalizeNonvoters(address: Address) -> None:
```

*Parameters:*

| Name  | Type            | Description                         |
|:------|:----------------|:------------------------------------|
| preps | List\[Address\] | addresses of P-Reps to be penalized |

*Event Log:*
[PenaltyImposed(Address,int,int)](#penaltyimposedaddressintint)

*Revision:* 15 ~

### setNetworkScore

Updates address of Network SCORE. Governance Only.

- Only SCORE owned by Governance can be Network SCORE

```
def setNetworkScore(role: str, address: Address) -> None:
```

*Parameters:*

| Name    | Type            | Description                                                                                        |
|:--------|:----------------|:---------------------------------------------------------------------------------------------------|
| role    | str             | type of Network SCORE. available `role` is [NETWORK_SCORE_TYPE](#networkscoretype)                 |
| address | List\[Address\] | (Optional from revision 17) address of Network SCORE. Do not pass `address` to clear Network SCORE |

*Event Log:*
- from revision 24
```
@eventlog(indexed=0)
def NetworkScoreSet(role: str, address: Address) -> None:
```

| Name    | Type     | Description                                                                        |
|:--------|:---------|:-----------------------------------------------------------------------------------|
| role    | str      | type of Network SCORE. available `role` is [NETWORK_SCORE_TYPE](#networkscoretype) |
| address | Address  | address of Network SCORE.                                                          |

*Revision:* 15 ~

### setRewardFundAllocation2

Updates allocation of reward fund. Governance only.

```
def setRewardFundAllocation2(values: List[NamedValue]) -> None:
```

*Parameters:*

| Name   | Type                              | Description                                                                                                |
|:-------|:----------------------------------|:-----------------------------------------------------------------------------------------------------------|
| values | List\[[NamedValue](#namedvalue)\] | available `name` is [REWARD_FUND_ALLOCATION_KEY](#rewardfundallocationkey)<br>sum of values must be 10,000 |

*Event Log:*
- from revision 24
```
@eventlog(indexed=0)
def RewardFundAllocationSet(type: str, value: int) -> None:
```

| Name  | Type | Description                                                                |
|:------|:-----|:---------------------------------------------------------------------------|
| name  | str  | available `name` is [REWARD_FUND_ALLOCATION_KEY](#rewardfundallocationkey) |
| value | int  | allocation value                                                           |

*Revision:* 24 ~

### setMinimumBond

Updates the minimum amount of bond that can earn minimum wage. Governance only.

```
def setMinimumBond(bond: int) -> None:
```

*Parameters:*

| Name                                                      | Type | Description                                                                          |
|:----------------------------------------------------------|:-----|:-------------------------------------------------------------------------------------|
| ${[REWARD_FUND_ALLOCATION_KEY](#rewardfundallocationkey)} | int  | rate allocated to the reward fund. (0 ~ 10,000). <br>sum of all rates must be 10,000 |

*Event Log:*

```
@eventlog(indexed=0)
def MinimumBondChanged(bond: int) -> None:
```

| Name | Type | Description         |
|:-----|:-----|:--------------------|
| bond | int  | minimum bond amount |

*Revision:* 24 ~

### initCommissionRate

Initializes commission rate parameters of the P-Rep.

- After initialization, `maxCommissionRate` and `maxCommissionChangeRate` can't be changed

```
def initCommissionRate(rate: int, maxRate: int, maxChangeRate: int) -> None:
```

*Parameters:*

| Name          | Type | Description                                               |
|:--------------|:-----|:----------------------------------------------------------|
| rate          | int  | commission rate. (0 ~ 10,000)                             |
| maxRate       | int  | maximum commission rate that P-Rep can configure          |
| maxChangeRate | int  | maximum rate of change of `commission rate` in one `Term` |

*Event Log:*

```
@eventlog(indexed=1)
def CommissionRateInitialized(address: Address, rate: int, maxRate: int, maxChangeRate: int) -> None:
```

| Name          | Type | Description                                               |
|:--------------|:-----|:----------------------------------------------------------|
| rate          | int  | commission rate                                           |
| maxRate       | int  | maximum commission rate that P-Rep can configure          |
| maxChangeRate | int  | maximum rate of change of `commission rate` in one `Term` |

*Revision:* 24 ~

### setCommissionRate

Updates commission rate of the P-Rep.

```
def setCommissionRate(rate: int) -> None:
```

*Parameters:*

| Name | Type | Description     |
|:-----|:-----|:----------------|
| rate | int  | commission rate |

*Event Log:*

```
@eventlog(indexed=1)
def CommissionRateChanged(address: Address, rate: int) -> None:
```

| Name | Type | Description     |
|:-----|:-----|:----------------|
| rate | int  | commission rate |

*Revision:* 24 ~

### setSlashingRates

Updates slashing rates of penalties. Governance only.

```
def setSlashingRates(rate: int) -> None:
```

*Parameters:*

| Name  | Type                              | Description                                                                        |
|:------|:----------------------------------|:-----------------------------------------------------------------------------------|
| rates | List\[[NamedValue](#namedvalue)\] | available `name` is [PENALTY_TYPE](#penaltytype)<br>range of `value` is 0 ~ 10,000 |

*Event Log:*

```
@eventlog(indexed=0)
def SlashingRateChangedV2(type: str, rate: int) -> None:
```

| Name | Type | Description   |
|:-----|:-----|:--------------|
| type | str  | penalty type  |
| rate | int  | slashing rate |

*Revision:* 24 ~

### requestUnjail

Requests unjail

```
def requestUnjail() -> None:
```

*Revision:* 25 ~

# BTP

## ReadOnly APIs

### getBTPNetworkTypeID

Returns BTP Network Type ID of the given `name`.

```
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

Returns a compressed public key for the P-Rep node address.

```
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

Opens a BTP Network. Governance only.

```
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

```
@eventlog(indexed=2)
def BTPNetworkTypeActivated(networkTypeName: str, networkTypeId: int) -> None:
```

| Name            | Type | Description                            |
|:----------------|:-----|:---------------------------------------|
| networkTypeName | str  | name of the activated BTP Network Type |
| networkTypeId   | int  | ID of the activated BTP Network Type   |

```
@eventlog(indexed=2)
def BTPNetworkOpened(networkTypeId: int, networkId: int) -> None:
```

| Name          | Type | Description                  |
|:--------------|:-----|:-----------------------------|
| networkTypeId | int  | ID of the BTP Network Type   |
| networkId     | int  | ID of the opened BTP Network |

*Revision:* 21 ~

### closeBTPNetwork

Closes a BTP Network. Governance only.

```
def closeBTPNetwork(id: int) -> None:
```

*Parameters:*

| Name | Type | Description    |
|:-----|:-----|:---------------|
| id   | int  | BTP Network ID |

*Event Log:*

```
@eventlog(indexed=2)
def BTPNetworkClosed(networkTypeId: int, networkId: int) -> None:
```

| Name          | Type | Description                  |
|:--------------|:-----|:-----------------------------|
| networkTypeId | int  | ID of the BTP Network Type   |
| networkId     | int  | ID of the closed BTP Network |

*Revision:* 21 ~

### sendBTPMessage

Sends a BTP message over the BTP Network. Only the owner of a BTP Network can send a BTP message.

```
def sendBTPMessage(networkId: int, message: bytes) -> None:
```

*Parameters:*

| Name      | Type  | Description    |
|:----------|:------|:---------------|
| networkId | str   | BTP Network ID |
| message   | bytes | BTP message    |

*Event Log:*

```
@eventlog(indexed=2)
def BTPMessage(networkId: int, messageSN: int) -> None:
```

| Name      | Type | Description                            |
|:----------|:-----|:---------------------------------------|
| networkId | int  | ID of the BTP Network                  |
| messageSN | int  | message sequence number in BTP Network |

*Revision:* 21 ~

### registerPRepNodePublicKey

Registers an initial public key for the P-Rep node address.

```
def registerPRepNodePublicKey(address: Address, pubKey: bytes) -> None:
```

*Parameters:*

| Name    | Type    | Description           |
|:--------|:--------|:----------------------|
| address | Address | address of P-Rep      |
| pubKey  | bytes   | compressed public key |

*Revision:* 21 ~

### setPRepNodePublicKey

Updates a public key for the P-Rep node address.

```
def setPRepNodePublicKey(pubKey: bytes) -> None:
```

*Parameters:*

| Name   | Type  | Description           |
|:-------|:------|:----------------------|
| pubKey | bytes | compressed public key |

*Revision:* 21 ~

# Types

## Value Types

| VALUE type                            | Description                                       | Example                                                            |
|:--------------------------------------|:--------------------------------------------------|:-------------------------------------------------------------------|
| <a id="T_ADDR_EOA">T_ADDR_EOA</a>     | "hx" + 40 digit HEX string                        | hxbe258ceb872e08851f1f59694dac2558708ece11                         |
| <a id="T_ADDR_SCORE">T_ADDR_SCORE</a> | "cx" + 40 digit HEX string                        | cxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32                         |
| <a id="T_HASH">T_HASH</a>             | "0x" + 64 digit HEX string                        | 0xc71303ef8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238 |
| <a id="T_INT">T_INT</a>               | "0x" + lowercase HEX string. No zero padding.     | 0xa                                                                |
| <a id="T_BOOL">T_BOOL</a>             | "0x1" for 'true', "0x0" for 'false'               | 0x1                                                                |
| <a id="T_BIN_DATA">T_BIN_DATA</a>     | "0x" + lowercase HEX string. Length must be even. | 0x34b2                                                             |

## StepCosts

| Key            | Value Type | Description                                                      |
|:---------------|:-----------|:-----------------------------------------------------------------|
| schema         | int        | Schema version (currently fixed at 1)                            |
| default        | int        | Default cost charged each time transaction is executed           |
| contractCall   | int        | Cost to call the smart contract function                         |
| contractCreate | int        | Cost to call the smart contract code generation function         |
| contractUpdate | int        | Cost to call the smart contract code update function             |
| contractSet    | int        | Cost to store the generated/updated smart contract code per byte |
| get            | int        | Cost to get values from the state database per byte              |
| getBase        | int        | Default cost charged each time `get` is called                   |
| set            | int        | Cost to set values newly in the state database per byte          |
| setBase        | int        | Default cost charged each time `set` is called                   |
| delete         | int        | Cost to delete values in the state database per byte             |
| deleteBase     | int        | Default cost charged each time `delete` is called                |
| input          | int        | Cost charged for input data included in transaction per byte     |
| log            | int        | Cost to emit event logs per byte                                 |
| logBase        | int        | Default cost charged each time `log` is called                   |
| apiCall        | int        | Cost charged for heavy API calls (e.g. hash functions)           |

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
| hasPublicKey           | bool       | (Optional) P-Rep has valid public keys for all active BTP Network type                                                                                                                                    |
| irep                   | int        | incentive rep used to calculate the reward for P-Rep<br>Limit: +- 20% of the previous value                                                                                                               |
| irepUpdateBlockHeight  | int        | block height when a P-Rep changed I-Rep value                                                                                                                                                             |
| lastHeight             | int        | latest block height at which the P-Rep's voting status changed                                                                                                                                            |
| name                   | str        | P-Rep name                                                                                                                                                                                                |
| nodeAddress            | str        | node Key for only consensus                                                                                                                                                                               |
| p2pEndpoint            | str        | network information used for connecting among P-Rep nodes                                                                                                                                                 |
| penalty                | int        | 0: None, 1: Disqualification, 2: Low Productivity, 3: Block Validation, 4: NonVote                                                                                                                        |
| power                  | int        | amount of power that a P-Rep receives from ICONist. (= min(`bonded`+`delegated`, `bonded` * 20))                                                                                                          |
| status                 | int        | 0: active, 1: unregistered                                                                                                                                                                                |
| totalBlocks            | int        | number of blocks that a P-Rep received when running as a Main P-Rep                                                                                                                                       |
| validatedBlocks        | int        | number of blocks that a P-Rep validated when running as a Main P-Rep                                                                                                                                      |
| website                | str        | P-Rep homepage URL                                                                                                                                                                                        |

## PRepSnapshot

| Key       | Value Type | Description                                                                                      |
|:----------|:-----------|:-------------------------------------------------------------------------------------------------|
| name      | str        | P-Rep name                                                                                       |
| address   | Address    | P-Rep address                                                                                    |
| delegated | int        | delegation amount that a P-Rep receives from ICONist                                             |
| power     | int        | amount of power that a P-Rep receives from ICONist. (= min(`bonded`+`delegated`, `bonded` * 20)) |

## PRepStats

| Key          | Value Type | Description                                                                      |
|:-------------|:-----------|:---------------------------------------------------------------------------------|
| fail         | int        | number of blocks that this PRep failed to validate until lastHeight              |
| failCont     | int        | number of consecutive blocks that this PRep failed to validate until lastHeight  |
| grade        | int        | 0: Main P-Rep, 1: Sub P-Rep, 2: P-Rep candidate                                  |
| lastHeight   | int        | Latest blockHeight when lastState change happened                                |
| lastState    | int        | 0: None, 1: Ready, 2: Success, 3: Failure                                        |
| owner        | Address    | PRep owner address                                                               |
| penalties    | int        | number of times that this PRep got penalized in the last 30 terms as a validator |
| realFail     | int        | number of blocks that this PRep failed to validate                               |
| realFailCont | int        | number of blocks that this PRep failed to validate consecutively                 |
| realTotal    | int        | number of blocks that this PRep was supposed to validate                         |
| status       | int        | 0: Active, 1: Unregistered, 2: Disqualified                                      |
| total        | int        | number of blocks that this PRep was supposed to validate until lastHeight        |

## ContractStatus

| KEY          | VALUE type        | Description                                                           |
|:-------------|:------------------|:----------------------------------------------------------------------|
| status       | str               | status of the contract. (`inactive`, `active`, `pending`, `rejected`) |
| deployTxHash | [T_HASH](#T_HASH) | TX Hash for deploy                                                    |
| auditTxHash  | [T_HASH](#T_HASH) | (Optional) TX Hash for audit                                          |

## DepositInfo

| KEY                  | VALUE type                  | Description                         |
|:---------------------|:----------------------------|:------------------------------------|
| availableDeposit     | [T_INT](#T_INT)             | available deposit amount            |
| availableVirtualStep | [T_INT](#T_INT)             | available virtual steps(deprecated) |
| deposits             | List\[[Deposit](#deposit)\] | remaining deposits                  |

## Deposit

### Deposit V1

| KEY               | VALUE type        | Description              |
|:------------------|:------------------|:-------------------------|
| id                | [T_HASH](#T_HASH) | ID of deposit            |
| depositRemain     | [T_INT](#T_INT)   | available deposit amount |
| depositUsed       | [T_INT](#T_INT)   | used deposit amount      |
| expires           | [T_INT](#T_INT)   | expiration block height  |
| virtualStepIssued | [T_INT](#T_INT)   | issued virtual steps     |
| virtualStepUsed   | [T_INT](#T_INT)   | used virtual steps       |

### Deposit V2

| KEY           | VALUE type      | Description              |
|:--------------|:----------------|:-------------------------|
| depositRemain | [T_INT](#T_INT) | available deposit amount |

## RewardFund

| KEY                                                       | VALUE type | Description                                                                                   |
|:----------------------------------------------------------|:-----------|:----------------------------------------------------------------------------------------------|
| Iglobal                                                   | int        | Iglobal amount                                                                                |
| ${[REWARD_FUND_ALLOCATION_KEY](#rewardfundallocationkey)} | int        | allocation rate.<br>sum of all rates is 10,000 if revision is less than 25, and 100 otherwise |

## NamedValue

| KEY   | VALUE type | Description |
|:------|:-----------|:------------|
| name  | str        | name        |       
| value | int        | value       |

# Event logs

## PenaltyImposed(Address,int,int)

```
@eventlog(indexed=1)
def PenaltyImposed(address: Address, status: int, penalty_type: int) -> None:
```

| Name         | Type    | Description                                                                                                                                                            |
|:-------------|:--------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| address      | Address | address of penalized P-Rep                                                                                                                                             |
| status       | int     | status of penalized P-Rep                                                                                                                                              |
| penalty_type | int     | type of penalty<br />1: Disqualification<br />2: Accumulated Validation Failure<br />3: Validation Failure<br />4: Missed Network Proposal Vote<br />5: Double signing |

# Predefined variables

## PENALTY_TYPE

| value                          | revision | Description                                    |
|:-------------------------------|:---------|:-----------------------------------------------|
| "disqualification"             | 6 ~      | P-Rep disqualification penalty                 |
| "accumulatedValidationFailure" | 6 ~      | accumulated block validation failure penalty   |
| "validationFailure"            | 6 ~      | block validation failure penalty               |
| "missedNetworkProposalVote"    | 6 ~      | missed Network Proposal vote penalty           |
| "doubleVote"                   | 25 ~     | submit multiple votes to same height and round |

## NETWORK_SCORE_TYPE

| value   | revision | Description             |
|:--------|:---------|:------------------------|
| "cps"   | 6 ~      | CPS Network SCORE       |
| "relay" | 6 ~      | BTP Relay NETWORK SCORE |

## REWARD_FUND_ALLOCATION_KEY

| value    | revision | Description                                |
|:---------|:---------|:-------------------------------------------|
| "Iprep"  | 13 ~     | key for P-Rep reward                       |
| "Ivoter" | 13 ~ 23  | key for Voter reward. Deprecated in IISS 4 |
| "Icps"   | 13 ~     | key for CPS reward                         |
| "Irelay" | 13 ~     | key for BTP relay reward                   |
| "Iwage"  | 23 ~     | key for P-Rep minimum wage                 |
