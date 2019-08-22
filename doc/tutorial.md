# Tutorial

## Introduction

This document describe how to setup the blockchain network with provided tools.
We provides two kinds of server.

| Name               | gochain | goloop |
|:-------------------|:-------:|:------:|
| JSON RPC           |   YES   |  YES   |
| Monitoring         |   YES   |  YES   |
| Multi-channel      |   NO    |  YES   |
| Management API,CLI |   NO    |  YES   |

## Tools

`gstool` makes keystore, genesis and genesis storage.
```bash
make gstool
```

## GOCHAIN

### Single node ( 1 Validator==GOD )

You need to make a configuration for the node.

```bash
./bin/gochain --save_key_store wallet.json --save config.json
```

It stores generated configuration(`config.json`) along with wallet keystore
(`wallet.json`). If you don't specify any password, it uses `gochain` as 
password of the keystore. You may apply more options while it generates.

Now, you may start the server with it.

```bash
./bin/gochain --config config.json
```

You may send transaction with the wallet (`wallet.json`) for initial balance
of your wallet.

This is single node configuration. If you want to make a network with multiple
nodes, you need to make own genesis and node configurations.

## GOLOOP

GOLOOP supports multiple chains can be configured with multiple nodes.
So, if you have running nodes, then you may start any chain in any time
if you have proper permission to it.

### Start server

Making configuration for the server

**Example**
* output key store : `ks0.json`
* output server configuration : `server0.json`
```bash
./bin/goloop server save --save_key_store ks0.json server0.json
```

You may apply more options for changing default configuration.
Now, you may start the server with the configuration

**Example**
* server configuration file : `server0.json`
```bash
./bin/goloop server -c server0.json start
```

If you want to use multiple nodes for a chain, you may start multiple servers in
hosts.

### Making genesis storage

We need to create key store for the god account (the account having all
permissions and assets at the initial stage). Of course, you may use
one of key stores of servers for it.

**Example**
* key store of god : `god.js`
```bash
./bin/gstool ks gen -o god.js
```

Then, we may make genesis for the chain with the key stores of validators.
You may use addresses of validators instead of key stores.

**Example**
* output file : `genesis.json`
* key store of god : `god.js`
* key stores of validators : `ks0.js` `ks1.js` `ks2.js` `ks3.js`
```bash
./bin/gstool gn gen -o genesis.json -g god.js ks0.js ks1.js ks2.js ks3.js
```

Then you may modify `genesis.json` according to your favour.
You may refer [Genesis Transaction](genesis_tx.md) for details.
And also you may use Genesis Template feature of
[Genesis Storage](genesis_storage.md). Then, make a storage from it.

**Example**
* output file : `gs.zip`
* genesis template or transaction file : `genesis.json`
```bash
./bin/gstool gs gen -o gs.zip -i genesis.json
```

You may check network ID of genesis storage with following command

**Example**
* genesis storage file : `gs.zip`
```bash
./bin/gstool gs info gs.zip
```

### Join the chain

Now, you need to join the chain. Before joining the chain, you need to specify
seed server address.

**Example**
* server configuration file : `server0.json`
* genesis storage file : `gs.zip`
* seed server host and port : `server0` `8080`
```bash
./bin/goloop -c server0.json chain join --genesis gs.zip --seed server0:8080
```

You may check whether it's successfully added with following command.

**Example**
* server configuration file : `server0.json`
```bash
./bin/goloop -c server0.json chain ls
```

### Start the chain

**Example**
* server configuration file : `server0.json`
* network ID   : `0xabcdef`
```bash
./bin/goloop -c server0.json chain start 0xabcdef
```
