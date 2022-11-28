# ICON Chain SCORE API

* [IISS](#iiss)
* [BTP 2.0](#btp-20)
  * [getBTPNetworkTypeID](#getbtpnetworktypeid)
  * [openBTPNetwork](#openbtpnetwork)
  * [closeBTPNetwork](#closebtpnetwork)
  * [sendBTPMessage](#sendbtpmessage)
  * [registerPRepNodePublicKey](#registerprepnodepublickey)
  * [setPRepNodePublicKey](#setprepnodepublickey)
  * [getPRepNodePublicKey](#getprepnodepublickey)

# IISS 3.0

## setStake
Stake some amount of ICX
```
void setStake(int value)
```
*Parameters:*
* `value` - the ICX amount of stake in loop

## getStake
Returns the stake status of a given address

```
dict getStake(Address address)

@External(readonly = true)
Map<String, Object> getStake(Address address)
```
*Parameters:*
* `address` - the address to query

*Returns:*

| Key                         | Value type | Description                               |
|:----------------------------|:-----------|:------------------------------------------|
| stake                       | int        | ICX amount of stake in loop               |
| unstakes                    | list       | Unstake information list                  |
| &nbsp;unstake               | int        | ICX amount of unstake in loop (indent)    |
| unstakes.unstake            | int        | ICX amount of unstake in loop (full path) |
| unstakes.unstakeBlockHeight | int        | BlockHeight when unstake will be done     |
| unstakes.remainingBlocks    | int        | Remaining blocks to `unstakeBlockHeight`  |

# BTP 2.0

## getBTPNetworkTypeID
Returns BTP Network Type ID of the specified `name`.
```java
int getBTPNetworkTypeID(String name)
```
*Parameters:*
* `name` - the name of BTP Network Type

*Returns:*
* an int value greater than 0 if BTP Network Type is active.
* an int value 0 if BTP Network Type is not active.
* an int value -1 if BTP Network Type is not supported.

## openBTPNetwork
Open a BTP Network.
```java
int openBTPNetwork(String networkTypeName, String name, Address owner)
```
*Parameters:*
* `networkTypeName` - the name of BTP Network Type
* `name` - the name of BTP Network
* `owner` - the owner of BTP Network

*Returns:*
* BTP Network ID or 0 if opening a BTP Network is failed

*Event Log:*
```java
@EventLog(indexed=2)
public void BTPNetworkTypeActivated(String networkTypeName, int networkTypeId) {}

@EventLog(indexed=2)
public void BTPNetworkOpened(int networkTypeId, int networkId) {}
```

## closeBTPNetwork
Close a BTP Network.
```java
void closeBTPNetwork(int id)
```

*Parameters:*
* `id` - the BTP Network ID

*Event Log:*
```java
@EventLog(indexed=2)
public void BTPNetworkClosed(int networkTypeId, int networkId) {}
```

## sendBTPMessage
Send a BTP message over the BTP Network. Only the owner of a BTP Network can send a BTP message.
```java
void sendBTPMessage(int networkId, byte[] message)
```
*Parameters:*
* `networkId` - the BTP Network ID
* `message` - BTP message

*Event Log:*
```java
@EventLog(indexed=2)
public void BTPMessage(int networkId, int messageSN) {}
```

## registerPRepNodePublicKey
Register an initial public key for the P-Rep node address.
```java
void registerPRepNodePublicKey(Address address, byte[] pubKey)
```

*Parameters:*
* `address` - the address of P-Rep
* `pubKey` - the public key

## setPRepNodePublicKey
Set a public key for the P-Rep node address.
```java
void setPRepNodePublicKey(byte[] pubKey)
```
*Parameters:*
* `pubKey` - the public key

## getPRepNodePublicKey
Returns a public key for the P-Rep node address.
```java
void getPRepNodePublicKey(Address address)
```
*Parameters:*
* `address` - the address of P-Rep

*Returns:*
* the public key or 'null' if the P-Rep does not have a public key
