# Goloop

## goloop

### Description
Goloop CLI

### Usage
` goloop `

### Child commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop chain

### Description
Manage chains

### Usage
` goloop chain TASK CID PARAM `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop chain backup

### Description
Start to backup the channel

### Usage
` goloop chain backup CID [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --manual |  | false | false |  Manual backup mode (just release database) |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain config

### Description
Configure chain

### Usage
` goloop chain config CID KEY VALUE [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --value |  | false |  |  use if value starts with '-'.
(if the third arg is used, this flag will be ignored) |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain genesis

### Description
Download chain genesis file

### Usage
` goloop chain genesis CID FILE `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain import

### Description
Start to import legacy database

### Usage
` goloop chain import CID [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --db_path |  | true |  |  Database path |
| --height |  | true | 0 |  Block Height |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain inspect

### Description
Inspect chain

### Usage
` goloop chain inspect CID [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --format, -f |  | false |  |  Format the output using the given Go template |
| --informal |  | false | false |  Inspect with informal data |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain join

### Description
Join chain

### Usage
` goloop chain join [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --auto_start |  | false | false |  Auto start |
| --channel |  | false |  |  Channel |
| --children_limit |  | false | -1 |  Maximum number of child connections (-1: uses system default value) |
| --concurrency |  | false | 1 |  Maximum number of executors to be used for concurrency |
| --db_type |  | false | goleveldb |  Name of database system(goleveldb, mapdb, rocksdb) |
| --default_wait_timeout |  | false | 0 |  Default wait timeout in milli-second (0: disable) |
| --genesis |  | false |  |  Genesis storage path |
| --genesis_template |  | false |  |  Genesis template directory or file |
| --max_block_tx_bytes |  | false | 0 |  Max size of transactions in a block |
| --max_wait_timeout |  | false | 0 |  Max wait timeout in milli-second (0: uses same value of default_wait_timeout) |
| --nephews_limit |  | false | -1 |  Maximum number of nephew connections (-1: uses system default value) |
| --node_cache |  | false | none |  Node cache (none,small,large) |
| --normal_tx_pool |  | false | 0 |  Size of normal transaction pool |
| --patch_tx_pool |  | false | 0 |  Size of patch transaction pool |
| --platform |  | false |  |  Name of service platform |
| --role |  | false | 3 |  [0:None, 1:Seed, 2:Validator, 3:Both] |
| --secure_aeads |  | false | chacha,aes128,aes256 |  Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string |
| --secure_suites |  | false | none,tls,ecdhe |  Supported Secure suites with order (none,tls,ecdhe) - Comma separated string |
| --seed |  | false |  |  List of trust-seed ip-port, Comma separated string |
| --tx_timeout |  | false | 0 |  Transaction timeout in milli-second (0: uses system default value) |
| --validate_tx_on_send |  | false | false |  Validate transaction on send |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain leave

### Description
Leave chain

### Usage
` goloop chain leave CID `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain ls

### Description
List chains

### Usage
` goloop chain ls `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain prune

### Description
Start to prune the database based on the height

### Usage
` goloop chain prune CID [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --db_type |  | false |  |  Database type(default:original database type) |
| --height |  | true | 0 |  Block Height |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain reset

### Description
Chain data reset

### Usage
` goloop chain reset CID [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --block_hash |  | false |  |  Hash of the block at the given height, If height is zero, shall be empty |
| --height |  | false | 0 |  Block Height |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain start

### Description
Chain start

### Usage
` goloop chain start CID `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain stop

### Description
Chain stop

### Usage
` goloop chain stop CID `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain verify

### Description
Chain data verify

### Usage
` goloop chain verify CID `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain backup](#goloop-chain-backup) |  Start to backup the channel |
| [goloop chain config](#goloop-chain-config) |  Configure chain |
| [goloop chain genesis](#goloop-chain-genesis) |  Download chain genesis file |
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain prune](#goloop-chain-prune) |  Start to prune the database based on the height |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop debug

### Description
DEBUG API

### Usage
` goloop debug `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --uri | GOLOOP_DEBUG_URI | true |  |  URI of DEBUG API |

### Child commands
|Command | Description|
|---|---|
| [goloop debug trace](#goloop-debug-trace) |  Get trace of the transaction |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop debug trace

### Description
Get trace of the transaction

### Usage
` goloop debug trace HASH `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --uri | GOLOOP_DEBUG_URI | true |  |  URI of DEBUG API |

### Parent command
|Command | Description|
|---|---|
| [goloop debug](#goloop-debug) |  DEBUG API |

### Related commands
|Command | Description|
|---|---|
| [goloop debug trace](#goloop-debug-trace) |  Get trace of the transaction |

## goloop gn

### Description
Genesis transaction manipulation

### Usage
` goloop gn `

### Child commands
|Command | Description|
|---|---|
| [goloop gn edit](#goloop-gn-edit) |  Edit genesis transaction |
| [goloop gn gen](#goloop-gn-gen) |  Generate genesis transaction |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop gn edit

### Description
Edit genesis transaction

### Usage
` goloop gn edit [genesis file] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --god, -g |  | false |  |  Address or keystore of GOD |
| --validator, -v |  | false | [] |  Address or keystore of Validator, [Validator...] |

### Parent command
|Command | Description|
|---|---|
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |

### Related commands
|Command | Description|
|---|---|
| [goloop gn edit](#goloop-gn-edit) |  Edit genesis transaction |
| [goloop gn gen](#goloop-gn-gen) |  Generate genesis transaction |

## goloop gn gen

### Description
Generate genesis transaction

### Usage
` goloop gn gen [address or keystore...] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c |  | false | [] |  Chain configuration |
| --fee |  | false | none |  Fee configuration (none,icon) |
| --god, -g |  | false |  |  Address or keystore of GOD |
| --out, -o |  | false | genesis.json |  Output file path |
| --supply, -s |  | false | 0x2961fff8ca4a62327800000 |  Total supply of the chain |
| --treasury, -t |  | false | hx1000000000000000000000000000000000000000 |  Treasury address |

### Parent command
|Command | Description|
|---|---|
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |

### Related commands
|Command | Description|
|---|---|
| [goloop gn edit](#goloop-gn-edit) |  Edit genesis transaction |
| [goloop gn gen](#goloop-gn-gen) |  Generate genesis transaction |

## goloop gs

### Description
Genesis storage manipulation

### Usage
` goloop gs `

### Child commands
|Command | Description|
|---|---|
| [goloop gs gen](#goloop-gs-gen) |  Create genesis storage from the template |
| [goloop gs info](#goloop-gs-info) |  Show genesis storage information |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop gs gen

### Description
Create genesis storage from the template

### Usage
` goloop gs gen `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --input, -i |  | false | genesis.json |  Input file or directory path |
| --out, -o |  | false | gs.zip |  Output file path |

### Parent command
|Command | Description|
|---|---|
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |

### Related commands
|Command | Description|
|---|---|
| [goloop gs gen](#goloop-gs-gen) |  Create genesis storage from the template |
| [goloop gs info](#goloop-gs-info) |  Show genesis storage information |

## goloop gs info

### Description
Show genesis storage information

### Usage
` goloop gs info genesis_storage.zip [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --cid_only, -c |  | false | false |  Showing chain ID only |
| --nid_only, -n |  | false | false |  Showing network ID only |

### Parent command
|Command | Description|
|---|---|
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |

### Related commands
|Command | Description|
|---|---|
| [goloop gs gen](#goloop-gs-gen) |  Create genesis storage from the template |
| [goloop gs info](#goloop-gs-info) |  Show genesis storage information |

## goloop ks

### Description
Keystore manipulation

### Usage
` goloop ks `

### Child commands
|Command | Description|
|---|---|
| [goloop ks gen](#goloop-ks-gen) |  Generate keystore |
| [goloop ks pubkey](#goloop-ks-pubkey) |  Generate publickey from keystore |
| [goloop ks verify](#goloop-ks-verify) |  Verify keystore with the password |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop ks gen

### Description
Generate keystore

### Usage
` goloop ks gen `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --out, -o |  | false | keystore.json |  Output file path |
| --password, -p |  | false | gochain |  Password for the keystore |

### Parent command
|Command | Description|
|---|---|
| [goloop ks](#goloop-ks) |  Keystore manipulation |

### Related commands
|Command | Description|
|---|---|
| [goloop ks gen](#goloop-ks-gen) |  Generate keystore |
| [goloop ks pubkey](#goloop-ks-pubkey) |  Generate publickey from keystore |
| [goloop ks verify](#goloop-ks-verify) |  Verify keystore with the password |

## goloop ks pubkey

### Description
Generate publickey from keystore

### Usage
` goloop ks pubkey `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --keystore, -k |  | false | keystore.json |  Keystore file path |
| --password, -p |  | false | gochain |  Password for the keystore |
| --secret, -s |  | false |  |  KeySecret file path |

### Parent command
|Command | Description|
|---|---|
| [goloop ks](#goloop-ks) |  Keystore manipulation |

### Related commands
|Command | Description|
|---|---|
| [goloop ks gen](#goloop-ks-gen) |  Generate keystore |
| [goloop ks pubkey](#goloop-ks-pubkey) |  Generate publickey from keystore |
| [goloop ks verify](#goloop-ks-verify) |  Verify keystore with the password |

## goloop ks verify

### Description
Verify keystore with the password

### Usage
` goloop ks verify `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --password, -p |  | false | gochain |  Password for the keystore |
| --secret, -s |  | false |  |  KeySecret file path |

### Parent command
|Command | Description|
|---|---|
| [goloop ks](#goloop-ks) |  Keystore manipulation |

### Related commands
|Command | Description|
|---|---|
| [goloop ks gen](#goloop-ks-gen) |  Generate keystore |
| [goloop ks pubkey](#goloop-ks-pubkey) |  Generate publickey from keystore |
| [goloop ks verify](#goloop-ks-verify) |  Verify keystore with the password |

## goloop rpc

### Description
JSON-RPC API

### Usage
` goloop rpc `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Child commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop rpc balance

### Description
GetBalance

### Usage
` goloop rpc balance ADDRESS [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --height |  | false | -1 |  BlockHeight |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc blockbyhash

### Description
GetBlockByHash

### Usage
` goloop rpc blockbyhash HASH `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc blockbyheight

### Description
GetBlockByHeight

### Usage
` goloop rpc blockbyheight HEIGHT `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc blockheaderbyheight

### Description
GetBlockHeaderByHeight

### Usage
` goloop rpc blockheaderbyheight HEIGHT `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc btpheader

### Description
GetBTPHeader

### Usage
` goloop rpc btpheader NETWORK_ID HEIGHT `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc btpmessages

### Description
GetBTPMessages

### Usage
` goloop rpc btpmessages NETWORK_ID HEIGHT `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc btpnetwork

### Description
GetBTPNetworkInfo

### Usage
` goloop rpc btpnetwork ID [HEIGHT] `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc btpnetworktype

### Description
GetBTPNetworkTypeInfo

### Usage
` goloop rpc btpnetworktype ID [HEIGHT] `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc btpproof

### Description
GetBTPProof

### Usage
` goloop rpc btpproof NETWORK_ID HEIGHT `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc btpsource

### Description
GetBTPSourceInformation

### Usage
` goloop rpc btpsource `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc call

### Description
Call

### Usage
` goloop rpc call [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --from |  | false |  |  FromAddress |
| --height |  | false | -1 |  BlockHeight |
| --method |  | false |  |  Name of the function to invoke in SCORE, if '--raw' used, will overwrite |
| --param |  | false | [] |  key=value, Function parameters, if '--raw' used, will overwrite |
| --params |  | false | [] |  raw json string or '@<json file>' or '-' for stdin for parameter JSON. it overrides raw one  |
| --raw |  | false |  |  call with 'data' using raw json file or json-string |
| --to |  | true |  |  ToAddress |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc databyhash

### Description
GetDataByHash

### Usage
` goloop rpc databyhash HASH `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc lastblock

### Description
GetLastBlock

### Usage
` goloop rpc lastblock `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc monitor

### Description
Monitor

### Usage
` goloop rpc monitor `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Child commands
|Command | Description|
|---|---|
| [goloop rpc monitor block](#goloop-rpc-monitor-block) |  MonitorBlock |
| [goloop rpc monitor btp](#goloop-rpc-monitor-btp) |  MonitorBTP |
| [goloop rpc monitor event](#goloop-rpc-monitor-event) |  MonitorEvent |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc monitor block

### Description
MonitorBlock

### Usage
` goloop rpc monitor block HEIGHT [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --filter |  | false | [] |  EventFilter raw json file or json string |
| --logs |  | false | false |  Includes logs |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc monitor block](#goloop-rpc-monitor-block) |  MonitorBlock |
| [goloop rpc monitor btp](#goloop-rpc-monitor-btp) |  MonitorBTP |
| [goloop rpc monitor event](#goloop-rpc-monitor-event) |  MonitorEvent |

## goloop rpc monitor btp

### Description
MonitorBTP

### Usage
` goloop rpc monitor btp HEIGHT [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --networkId |  | false |  |  BTP Network ID |
| --proof_flag |  | false | false |  Includes proof |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc monitor block](#goloop-rpc-monitor-block) |  MonitorBlock |
| [goloop rpc monitor btp](#goloop-rpc-monitor-btp) |  MonitorBTP |
| [goloop rpc monitor event](#goloop-rpc-monitor-event) |  MonitorEvent |

## goloop rpc monitor event

### Description
MonitorEvent

### Usage
` goloop rpc monitor event HEIGHT [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --addr |  | false |  |  SCORE Address |
| --data |  | false | [] |  Not indexed Arguments of Event, comma-separated string |
| --event |  | false |  |  Signature of Event |
| --indexed |  | false | [] |  Indexed Arguments of Event, comma-separated string |
| --logs |  | false | false |  Includes logs |
| --raw |  | false |  |  EventFilter raw json file or json-string |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc monitor block](#goloop-rpc-monitor-block) |  MonitorBlock |
| [goloop rpc monitor btp](#goloop-rpc-monitor-btp) |  MonitorBTP |
| [goloop rpc monitor event](#goloop-rpc-monitor-event) |  MonitorEvent |

## goloop rpc proofforevents

### Description
GetProofForEvents

### Usage
` goloop rpc proofforevents BLOCK_HASH TX_INDEX EVENT_INDEXES `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc proofforresult

### Description
GetProofForResult

### Usage
` goloop rpc proofforresult HASH INDEX `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc raw

### Description
Rpc with raw json file

### Usage
` goloop rpc raw FILE `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc scoreapi

### Description
GetScoreApi

### Usage
` goloop rpc scoreapi ADDRESS [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --height |  | false | -1 |  BlockHeight |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc scorestatus

### Description
Get status of the smart contract

### Usage
` goloop rpc scorestatus ADDRESS [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --height |  | false | -1 |  BlockHeight |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc sendtx

### Description
SendTransaction

### Usage
` goloop rpc sendtx `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --estimate | GOLOOP_RPC_ESTIMATE | false | false |  Just estimate steps for the tx |
| --key_password | GOLOOP_RPC_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET | false |  |  Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE | true |  |  KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID | true |  |  Network ID |
| --save | GOLOOP_RPC_SAVE | false |  |  Store transaction to the file |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | false | 0 |  StepLimit |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Child commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature |
| [goloop rpc sendtx raw2](#goloop-rpc-sendtx-raw2) |  Send transaction with json file overwriting timestamp and signature |
| [goloop rpc sendtx raw3](#goloop-rpc-sendtx-raw3) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc sendtx call

### Description
SmartContract Call Transaction

### Usage
` goloop rpc sendtx call [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --method |  | true |  |  Name of the function to invoke in SCORE, if '--raw' used, will overwrite |
| --param |  | false | [] |  key=value, Function parameters, if '--raw' used, will overwrite |
| --params |  | false |  |  raw json string or '@<json file>' or '-' for stdin for parameter JSON, it overrides raw one |
| --raw |  | false |  |  call with 'data' using raw json file or json-string |
| --to |  | true |  |  ToAddress |
| --value |  | false |  |  Value of transfer |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --estimate | GOLOOP_RPC_ESTIMATE | false | false |  Just estimate steps for the tx |
| --key_password | GOLOOP_RPC_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET | false |  |  Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE | true |  |  KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID | true |  |  Network ID |
| --save | GOLOOP_RPC_SAVE | false |  |  Store transaction to the file |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | false | 0 |  StepLimit |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature |
| [goloop rpc sendtx raw2](#goloop-rpc-sendtx-raw2) |  Send transaction with json file overwriting timestamp and signature |
| [goloop rpc sendtx raw3](#goloop-rpc-sendtx-raw3) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc sendtx deploy

### Description
Deploy Transaction

### Usage
` goloop rpc sendtx deploy SCORE_ZIP_FILE [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --content_type |  | false | application/zip |  Mime-type of the content |
| --param |  | false | [] |  key=value, Function parameters will be delivered to on_install() or on_update() |
| --params |  | false |  |  raw json string or '@<json file>' or '-' for stdin for parameter JSON |
| --to |  | false | cx0000000000000000000000000000000000000000 |  ToAddress |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --estimate | GOLOOP_RPC_ESTIMATE | false | false |  Just estimate steps for the tx |
| --key_password | GOLOOP_RPC_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET | false |  |  Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE | true |  |  KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID | true |  |  Network ID |
| --save | GOLOOP_RPC_SAVE | false |  |  Store transaction to the file |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | false | 0 |  StepLimit |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature |
| [goloop rpc sendtx raw2](#goloop-rpc-sendtx-raw2) |  Send transaction with json file overwriting timestamp and signature |
| [goloop rpc sendtx raw3](#goloop-rpc-sendtx-raw3) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc sendtx raw

### Description
Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature

### Usage
` goloop rpc sendtx raw FILE `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --estimate | GOLOOP_RPC_ESTIMATE | false | false |  Just estimate steps for the tx |
| --key_password | GOLOOP_RPC_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET | false |  |  Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE | true |  |  KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID | true |  |  Network ID |
| --save | GOLOOP_RPC_SAVE | false |  |  Store transaction to the file |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | false | 0 |  StepLimit |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature |
| [goloop rpc sendtx raw2](#goloop-rpc-sendtx-raw2) |  Send transaction with json file overwriting timestamp and signature |
| [goloop rpc sendtx raw3](#goloop-rpc-sendtx-raw3) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc sendtx raw2

### Description
Send transaction with json file overwriting timestamp and signature

### Usage
` goloop rpc sendtx raw2 FILE `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --estimate | GOLOOP_RPC_ESTIMATE | false | false |  Just estimate steps for the tx |
| --key_password | GOLOOP_RPC_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET | false |  |  Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE | true |  |  KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID | true |  |  Network ID |
| --save | GOLOOP_RPC_SAVE | false |  |  Store transaction to the file |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | false | 0 |  StepLimit |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature |
| [goloop rpc sendtx raw2](#goloop-rpc-sendtx-raw2) |  Send transaction with json file overwriting timestamp and signature |
| [goloop rpc sendtx raw3](#goloop-rpc-sendtx-raw3) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc sendtx raw3

### Description
Send transaction with json file

### Usage
` goloop rpc sendtx raw3 FILE `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --estimate | GOLOOP_RPC_ESTIMATE | false | false |  Just estimate steps for the tx |
| --key_password | GOLOOP_RPC_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET | false |  |  Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE | true |  |  KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID | true |  |  Network ID |
| --save | GOLOOP_RPC_SAVE | false |  |  Store transaction to the file |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | false | 0 |  StepLimit |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature |
| [goloop rpc sendtx raw2](#goloop-rpc-sendtx-raw2) |  Send transaction with json file overwriting timestamp and signature |
| [goloop rpc sendtx raw3](#goloop-rpc-sendtx-raw3) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc sendtx transfer

### Description
Coin Transfer Transaction

### Usage
` goloop rpc sendtx transfer [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --message |  | false |  |  Message |
| --to |  | true |  |  ToAddress |
| --value |  | true |  |  Value |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --estimate | GOLOOP_RPC_ESTIMATE | false | false |  Just estimate steps for the tx |
| --key_password | GOLOOP_RPC_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET | false |  |  Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE | true |  |  KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID | true |  |  Network ID |
| --save | GOLOOP_RPC_SAVE | false |  |  Store transaction to the file |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | false | 0 |  StepLimit |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file filling nid,version,stepLimit,from and overwriting timestamp and signature |
| [goloop rpc sendtx raw2](#goloop-rpc-sendtx-raw2) |  Send transaction with json file overwriting timestamp and signature |
| [goloop rpc sendtx raw3](#goloop-rpc-sendtx-raw3) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc totalsupply

### Description
GetTotalSupply

### Usage
` goloop rpc totalsupply [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --height |  | false | -1 |  BlockHeight |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc txbyhash

### Description
GetTransactionByHash

### Usage
` goloop rpc txbyhash HASH `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc txresult

### Description
GetTransactionResult

### Usage
` goloop rpc txresult HASH `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc votesbyheight

### Description
GetVotesByHeight

### Usage
` goloop rpc votesbyheight HEIGHT `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | false |  JSON-RPC Response with detail information |
| --debug_uri | GOLOOP_RPC_DEBUG_URI | false |  |  URI of JSON-RPC Debug API |
| --uri | GOLOOP_RPC_URI | true |  |  URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc btpheader](#goloop-rpc-btpheader) |  GetBTPHeader |
| [goloop rpc btpmessages](#goloop-rpc-btpmessages) |  GetBTPMessages |
| [goloop rpc btpnetwork](#goloop-rpc-btpnetwork) |  GetBTPNetworkInfo |
| [goloop rpc btpnetworktype](#goloop-rpc-btpnetworktype) |  GetBTPNetworkTypeInfo |
| [goloop rpc btpproof](#goloop-rpc-btpproof) |  GetBTPProof |
| [goloop rpc btpsource](#goloop-rpc-btpsource) |  GetBTPSourceInformation |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforevents](#goloop-rpc-proofforevents) |  GetProofForEvents |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc scorestatus](#goloop-rpc-scorestatus) |  Get status of the smart contract |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop server

### Description
Server management

### Usage
` goloop server `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --backup_dir | GOLOOP_BACKUP_DIR | false |  |  Node backup directory (default: [node_dir]/backup |
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --console_level | GOLOOP_CONSOLE_LEVEL | false | trace |  Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket | GOLOOP_EE_SOCKET | false |  |  Execution engine socket path |
| --engines | GOLOOP_ENGINES | false | python |  Execution engines, comma-separated (python,java) |
| --key_password | GOLOOP_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_plugin | GOLOOP_KEY_PLUGIN | false |  |  KeyPlugin file for wallet |
| --key_plugin_options | GOLOOP_KEY_PLUGIN_OPTIONS | false | [] |  KeyPlugin options |
| --key_secret | GOLOOP_KEY_SECRET | false |  |  Secret (password) file for KeyStore |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --log_forwarder_address | GOLOOP_LOG_FORWARDER_ADDRESS | false |  |  LogForwarder address |
| --log_forwarder_level | GOLOOP_LOG_FORWARDER_LEVEL | false | info |  LogForwarder level |
| --log_forwarder_name | GOLOOP_LOG_FORWARDER_NAME | false |  |  LogForwarder name |
| --log_forwarder_options | GOLOOP_LOG_FORWARDER_OPTIONS | false | [] |  LogForwarder options, comma-separated 'key=value' |
| --log_forwarder_vendor | GOLOOP_LOG_FORWARDER_VENDOR | false |  |  LogForwarder vendor (fluentd,logstash) |
| --log_level | GOLOOP_LOG_LEVEL | false | debug |  Global log level (trace,debug,info,warn,error,fatal,panic) |
| --log_writer_compress | GOLOOP_LOG_WRITER_COMPRESS | false | false |  Use gzip on rotated log file |
| --log_writer_filename | GOLOOP_LOG_WRITER_FILENAME | false |  |  Log filename (rotated files resides in same directory) |
| --log_writer_localtime | GOLOOP_LOG_WRITER_LOCALTIME | false | false |  Use localtime on rotated log file instead of UTC |
| --log_writer_maxage | GOLOOP_LOG_WRITER_MAXAGE | false | 0 |  Maximum age of log file in day |
| --log_writer_maxbackups | GOLOOP_LOG_WRITER_MAXBACKUPS | false | 0 |  Maximum number of backups |
| --log_writer_maxsize | GOLOOP_LOG_WRITER_MAXSIZE | false | 100 |  Maximum log file size in MiB |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory (default: [configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | false |  |  Node Command Line Interface socket path (default: [node_dir]/cli.sock) |
| --p2p | GOLOOP_P2P | false | 127.0.0.1:8080 |  Advertise ip-port of P2P |
| --p2p_listen | GOLOOP_P2P_LISTEN | false |  |  Listen ip-port of P2P |
| --rpc_addr | GOLOOP_RPC_ADDR | false | :9080 |  Listen ip-port of JSON-RPC |
| --rpc_dump | GOLOOP_RPC_DUMP | false | false |  JSON-RPC Request, Response Dump flag |

### Child commands
|Command | Description|
|---|---|
| [goloop server save](#goloop-server-save) |  Save configuration |
| [goloop server start](#goloop-server-start) |  Start server |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop server save

### Description
Save configuration

### Usage
` goloop server save [file] [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --save_key_store |  | false |  |  KeyStore File path to save |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --backup_dir | GOLOOP_BACKUP_DIR | false |  |  Node backup directory (default: [node_dir]/backup |
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --console_level | GOLOOP_CONSOLE_LEVEL | false | trace |  Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket | GOLOOP_EE_SOCKET | false |  |  Execution engine socket path |
| --engines | GOLOOP_ENGINES | false | python |  Execution engines, comma-separated (python,java) |
| --key_password | GOLOOP_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_plugin | GOLOOP_KEY_PLUGIN | false |  |  KeyPlugin file for wallet |
| --key_plugin_options | GOLOOP_KEY_PLUGIN_OPTIONS | false | [] |  KeyPlugin options |
| --key_secret | GOLOOP_KEY_SECRET | false |  |  Secret (password) file for KeyStore |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --log_forwarder_address | GOLOOP_LOG_FORWARDER_ADDRESS | false |  |  LogForwarder address |
| --log_forwarder_level | GOLOOP_LOG_FORWARDER_LEVEL | false | info |  LogForwarder level |
| --log_forwarder_name | GOLOOP_LOG_FORWARDER_NAME | false |  |  LogForwarder name |
| --log_forwarder_options | GOLOOP_LOG_FORWARDER_OPTIONS | false | [] |  LogForwarder options, comma-separated 'key=value' |
| --log_forwarder_vendor | GOLOOP_LOG_FORWARDER_VENDOR | false |  |  LogForwarder vendor (fluentd,logstash) |
| --log_level | GOLOOP_LOG_LEVEL | false | debug |  Global log level (trace,debug,info,warn,error,fatal,panic) |
| --log_writer_compress | GOLOOP_LOG_WRITER_COMPRESS | false | false |  Use gzip on rotated log file |
| --log_writer_filename | GOLOOP_LOG_WRITER_FILENAME | false |  |  Log filename (rotated files resides in same directory) |
| --log_writer_localtime | GOLOOP_LOG_WRITER_LOCALTIME | false | false |  Use localtime on rotated log file instead of UTC |
| --log_writer_maxage | GOLOOP_LOG_WRITER_MAXAGE | false | 0 |  Maximum age of log file in day |
| --log_writer_maxbackups | GOLOOP_LOG_WRITER_MAXBACKUPS | false | 0 |  Maximum number of backups |
| --log_writer_maxsize | GOLOOP_LOG_WRITER_MAXSIZE | false | 100 |  Maximum log file size in MiB |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory (default: [configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | false |  |  Node Command Line Interface socket path (default: [node_dir]/cli.sock) |
| --p2p | GOLOOP_P2P | false | 127.0.0.1:8080 |  Advertise ip-port of P2P |
| --p2p_listen | GOLOOP_P2P_LISTEN | false |  |  Listen ip-port of P2P |
| --rpc_addr | GOLOOP_RPC_ADDR | false | :9080 |  Listen ip-port of JSON-RPC |
| --rpc_dump | GOLOOP_RPC_DUMP | false | false |  JSON-RPC Request, Response Dump flag |

### Parent command
|Command | Description|
|---|---|
| [goloop server](#goloop-server) |  Server management |

### Related commands
|Command | Description|
|---|---|
| [goloop server save](#goloop-server-save) |  Save configuration |
| [goloop server start](#goloop-server-start) |  Start server |

## goloop server start

### Description
Start server

### Usage
` goloop server start [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --cpuprofile |  | false |  |  CPU Profiling data file |
| --memprofile |  | false |  |  Memory Profiling data file |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --backup_dir | GOLOOP_BACKUP_DIR | false |  |  Node backup directory (default: [node_dir]/backup |
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --console_level | GOLOOP_CONSOLE_LEVEL | false | trace |  Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket | GOLOOP_EE_SOCKET | false |  |  Execution engine socket path |
| --engines | GOLOOP_ENGINES | false | python |  Execution engines, comma-separated (python,java) |
| --key_password | GOLOOP_KEY_PASSWORD | false |  |  Password for the KeyStore file |
| --key_plugin | GOLOOP_KEY_PLUGIN | false |  |  KeyPlugin file for wallet |
| --key_plugin_options | GOLOOP_KEY_PLUGIN_OPTIONS | false | [] |  KeyPlugin options |
| --key_secret | GOLOOP_KEY_SECRET | false |  |  Secret (password) file for KeyStore |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --log_forwarder_address | GOLOOP_LOG_FORWARDER_ADDRESS | false |  |  LogForwarder address |
| --log_forwarder_level | GOLOOP_LOG_FORWARDER_LEVEL | false | info |  LogForwarder level |
| --log_forwarder_name | GOLOOP_LOG_FORWARDER_NAME | false |  |  LogForwarder name |
| --log_forwarder_options | GOLOOP_LOG_FORWARDER_OPTIONS | false | [] |  LogForwarder options, comma-separated 'key=value' |
| --log_forwarder_vendor | GOLOOP_LOG_FORWARDER_VENDOR | false |  |  LogForwarder vendor (fluentd,logstash) |
| --log_level | GOLOOP_LOG_LEVEL | false | debug |  Global log level (trace,debug,info,warn,error,fatal,panic) |
| --log_writer_compress | GOLOOP_LOG_WRITER_COMPRESS | false | false |  Use gzip on rotated log file |
| --log_writer_filename | GOLOOP_LOG_WRITER_FILENAME | false |  |  Log filename (rotated files resides in same directory) |
| --log_writer_localtime | GOLOOP_LOG_WRITER_LOCALTIME | false | false |  Use localtime on rotated log file instead of UTC |
| --log_writer_maxage | GOLOOP_LOG_WRITER_MAXAGE | false | 0 |  Maximum age of log file in day |
| --log_writer_maxbackups | GOLOOP_LOG_WRITER_MAXBACKUPS | false | 0 |  Maximum number of backups |
| --log_writer_maxsize | GOLOOP_LOG_WRITER_MAXSIZE | false | 100 |  Maximum log file size in MiB |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory (default: [configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | false |  |  Node Command Line Interface socket path (default: [node_dir]/cli.sock) |
| --p2p | GOLOOP_P2P | false | 127.0.0.1:8080 |  Advertise ip-port of P2P |
| --p2p_listen | GOLOOP_P2P_LISTEN | false |  |  Listen ip-port of P2P |
| --rpc_addr | GOLOOP_RPC_ADDR | false | :9080 |  Listen ip-port of JSON-RPC |
| --rpc_dump | GOLOOP_RPC_DUMP | false | false |  JSON-RPC Request, Response Dump flag |

### Parent command
|Command | Description|
|---|---|
| [goloop server](#goloop-server) |  Server management |

### Related commands
|Command | Description|
|---|---|
| [goloop server save](#goloop-server-save) |  Save configuration |
| [goloop server start](#goloop-server-start) |  Start server |

## goloop stats

### Description
Display a live streams of chains metric-statistics

### Usage
` goloop stats `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --interval | GOLOOP_INTERVAL | false | 1 |  Pull interval |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --no-stream | GOLOOP_NO-STREAM | false | false |  Only pull the first metric-statistics |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop system

### Description
System info

### Usage
` goloop system `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop system backup](#goloop-system-backup) |  Manage stored backups |
| [goloop system config](#goloop-system-config) |  Configure system |
| [goloop system info](#goloop-system-info) |  Get system information |
| [goloop system restore](#goloop-system-restore) |  Restore chain from a backup |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop system backup

### Description
Manage stored backups

### Usage
` goloop system backup `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop system backup ls](#goloop-system-backup-ls) |  List current backups |

### Parent command
|Command | Description|
|---|---|
| [goloop system](#goloop-system) |  System info |

### Related commands
|Command | Description|
|---|---|
| [goloop system backup](#goloop-system-backup) |  Manage stored backups |
| [goloop system config](#goloop-system-config) |  Configure system |
| [goloop system info](#goloop-system-info) |  Get system information |
| [goloop system restore](#goloop-system-restore) |  Restore chain from a backup |

## goloop system backup ls

### Description
List current backups

### Usage
` goloop system backup ls `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c |  | false |  |  Parsing configuration file |
| --key_store |  | false |  |  KeyStore file for wallet |
| --node_dir |  | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop system backup](#goloop-system-backup) |  Manage stored backups |

### Related commands
|Command | Description|
|---|---|
| [goloop system backup ls](#goloop-system-backup-ls) |  List current backups |

## goloop system config

### Description
Configure system

### Usage
` goloop system config KEY VALUE `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop system](#goloop-system) |  System info |

### Related commands
|Command | Description|
|---|---|
| [goloop system backup](#goloop-system-backup) |  Manage stored backups |
| [goloop system config](#goloop-system-config) |  Configure system |
| [goloop system info](#goloop-system-info) |  Get system information |
| [goloop system restore](#goloop-system-restore) |  Restore chain from a backup |

## goloop system info

### Description
Get system information

### Usage
` goloop system info [flags] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --format, -f |  | false |  |  Format the output using the given Go template |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop system](#goloop-system) |  System info |

### Related commands
|Command | Description|
|---|---|
| [goloop system backup](#goloop-system-backup) |  Manage stored backups |
| [goloop system config](#goloop-system-config) |  Configure system |
| [goloop system info](#goloop-system-info) |  Get system information |
| [goloop system restore](#goloop-system-restore) |  Restore chain from a backup |

## goloop system restore

### Description
Restore chain from a backup

### Usage
` goloop system restore `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop system restore start](#goloop-system-restore-start) |  Start to restore the specified backup |
| [goloop system restore status](#goloop-system-restore-status) |  Get restore status |
| [goloop system restore stop](#goloop-system-restore-stop) |  Stop current restoring job |

### Parent command
|Command | Description|
|---|---|
| [goloop system](#goloop-system) |  System info |

### Related commands
|Command | Description|
|---|---|
| [goloop system backup](#goloop-system-backup) |  Manage stored backups |
| [goloop system config](#goloop-system-config) |  Configure system |
| [goloop system info](#goloop-system-info) |  Get system information |
| [goloop system restore](#goloop-system-restore) |  Restore chain from a backup |

## goloop system restore start

### Description
Start to restore the specified backup

### Usage
` goloop system restore start [NAME] `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --overwrite |  | false | false |  Overwrite existing chain |

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c |  | false |  |  Parsing configuration file |
| --key_store |  | false |  |  KeyStore file for wallet |
| --node_dir |  | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop system restore](#goloop-system-restore) |  Restore chain from a backup |

### Related commands
|Command | Description|
|---|---|
| [goloop system restore start](#goloop-system-restore-start) |  Start to restore the specified backup |
| [goloop system restore status](#goloop-system-restore-status) |  Get restore status |
| [goloop system restore stop](#goloop-system-restore-stop) |  Stop current restoring job |

## goloop system restore status

### Description
Get restore status

### Usage
` goloop system restore status `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c |  | false |  |  Parsing configuration file |
| --key_store |  | false |  |  KeyStore file for wallet |
| --node_dir |  | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop system restore](#goloop-system-restore) |  Restore chain from a backup |

### Related commands
|Command | Description|
|---|---|
| [goloop system restore start](#goloop-system-restore-start) |  Start to restore the specified backup |
| [goloop system restore status](#goloop-system-restore-status) |  Get restore status |
| [goloop system restore stop](#goloop-system-restore-stop) |  Stop current restoring job |

## goloop system restore stop

### Description
Stop current restoring job

### Usage
` goloop system restore stop `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c |  | false |  |  Parsing configuration file |
| --key_store |  | false |  |  KeyStore file for wallet |
| --node_dir |  | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop system restore](#goloop-system-restore) |  Restore chain from a backup |

### Related commands
|Command | Description|
|---|---|
| [goloop system restore start](#goloop-system-restore-start) |  Start to restore the specified backup |
| [goloop system restore status](#goloop-system-restore-status) |  Get restore status |
| [goloop system restore stop](#goloop-system-restore-stop) |  Stop current restoring job |

## goloop user

### Description
User management

### Usage
` goloop user `

### Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop user add](#goloop-user-add) |  Add user |
| [goloop user ls](#goloop-user-ls) |  List users |
| [goloop user rm](#goloop-user-rm) |  Remove user |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

## goloop user add

### Description
Add user

### Usage
` goloop user add ADDRESS `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop user](#goloop-user) |  User management |

### Related commands
|Command | Description|
|---|---|
| [goloop user add](#goloop-user-add) |  Add user |
| [goloop user ls](#goloop-user-ls) |  List users |
| [goloop user rm](#goloop-user-rm) |  Remove user |

## goloop user ls

### Description
List users

### Usage
` goloop user ls `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop user](#goloop-user) |  User management |

### Related commands
|Command | Description|
|---|---|
| [goloop user add](#goloop-user-add) |  Add user |
| [goloop user ls](#goloop-user-ls) |  List users |
| [goloop user rm](#goloop-user-rm) |  Remove user |

## goloop user rm

### Description
Remove user

### Usage
` goloop user rm ADDRESS `

### Inherited Options
|Name,shorthand | Environment Variable | Required | Default | Description|
|---|---|---|---|---|
| --config, -c | GOLOOP_CONFIG | false |  |  Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE | false |  |  KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR | false |  |  Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK | true |  |  Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop user](#goloop-user) |  User management |

### Related commands
|Command | Description|
|---|---|
| [goloop user add](#goloop-user-add) |  Add user |
| [goloop user ls](#goloop-user-ls) |  List users |
| [goloop user rm](#goloop-user-rm) |  Remove user |

## goloop version

### Description
Print goloop version

### Usage
` goloop version `

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
| [goloop debug](#goloop-debug) |  DEBUG API |
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

