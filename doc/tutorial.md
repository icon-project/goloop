# Tutorial

## Introduction

This document describes how to setup a blockchain network with the provided tools.
We provides two types of servers, `gochain` and `goloop`.

| Name                | gochain | goloop |
|:--------------------|:-------:|:------:|
| JSON RPC            |   YES   |  YES   |
| Monitoring          |   YES   |  YES   |
| Multi-channel       |   NO    |  YES   |
| Management API, CLI |   NO    |  YES   |

## GOCHAIN

### Single node ( 1 Validator == GOD )

You need to make a configuration for the node.

```bash
./bin/gochain --save_key_store wallet.json --save config.json
```

It generates a configuration file, `config.json`, along with a wallet keystore file, `wallet.json`.
If you don't specify any password, it uses `gochain` as a default password for the keystore.
You may apply more options while it generates.
Please run `./bin/gochain --help` for more information.

Now, you may start the server with it.

```bash
./bin/gochain --config config.json
```

In another terminal, you can test if everything is working fine by making a JSON RPC request using GOLOOP binary.

```bash
./bin/goloop rpc lastblock --uri http://127.0.0.1:9080/api/v3
```

You may send transactions with the wallet, `wallet.json`, for the initial balance of other wallets.

This is a single node configuration. If you want to make a network with multiple nodes,
you need to make your own genesis and node configurations.

## GOLOOP

GOLOOP supports multiple chains that can be configured with multiple nodes.
This means, when you are running nodes, you may start any chain in any time
if you have proper permission to do it.

### Start the server

First, create a configuration for the server.

**Example**
* output keystore : `ks0.json`
* output server configuration : `server0.json`
```bash
./bin/goloop server save --save_key_store ks0.json server0.json
```

You may apply more options to change the default configuration.
Now, you can start the server with the configuration.

**Example**
* server configuration file : `server0.json`
```bash
./bin/goloop server -c server0.json start
```
**[NOTE]** You may need to activate the Python virtual environment before starting the server.

If you want to use multiple nodes for a chain, you need to start multiple servers in hosts.

### Create genesis storage

You need to create a keystore for the god account (the account having all permissions and assets at the initial stage).
Of course, you may use one of keystores of servers for it.

**Example**
* keystore of god : `god.json`
```bash
./bin/goloop ks gen -o god.json
```

Then, you may create the genesis for the chain with keystores of validators.
You may use addresses of validators instead of keystores.

**Example**
* output file : `genesis.json`
* keystore of god : `god.json`
* keystores of validators : `ks0.json` `ks1.json` `ks2.json` `ks3.json`
```bash
./bin/goloop gn gen -o genesis.json -g god.json ks0.json ks1.json ks2.json ks3.json
```

Then you may modify `genesis.json` according to your preferences.
You may refer [Genesis Transaction](genesis_tx.md) for more details.
And also you may use Genesis Template feature of [Genesis Storage](genesis_storage.md),
then create genesis storage from it.

**Example**
* output file : `gs.zip`
* genesis template or transaction file : `genesis.json`
```bash
./bin/goloop gs gen -o gs.zip -i genesis.json
```

You can check the network ID of genesis storage with the following command.

**Example**
* genesis storage file : `gs.zip`
```bash
./bin/goloop gs info gs.zip
```

### Join the chain

Now, you need to join the chain.
Before joining the chain, you need to specify the seed server address.
From another terminal, run the following command to configure the seed server.

**Example**
* server configuration file : `server0.json`
* genesis storage file : `gs.zip`
* seed server host and port : `server0:8080`
```bash
./bin/goloop -c server0.json chain join --genesis gs.zip --seed server0:8080
```

You can check whether it's successfully added with the following command.

**Example**
* server configuration file : `server0.json`
```bash
./bin/goloop -c server0.json chain ls
```

### Start the chain

Now you can start the chain with the following command.

**Example**
* server configuration file : `server0.json`
* network ID : `0xabcdef`
```bash
./bin/goloop -c server0.json chain start 0xabcdef
```
