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
` goloop chain `

### Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
` goloop chain backup CID `

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
` goloop chain config CID KEY VALUE `

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --db_path |  |  | Database path |
| --height |  | 0 | Block Height |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --format, -f |  |  | Format the output using the given Go template |
| --informal |  | false | Inspect with informal data |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --channel |  |  | Channel |
| --concurrency |  | 1 | Maximum number of executors to be used for concurrency |
| --db_type |  | goleveldb | Name of database system(*badgerdb, goleveldb, boltdb, mapdb) |
| --default_wait_timeout |  | 0 | Default wait timeout in milli-second (0: disable) |
| --genesis |  |  | Genesis storage path |
| --genesis_template |  |  | Genesis template directory or file |
| --max_block_tx_bytes |  | 0 | Max size of transactions in a block |
| --max_wait_timeout |  | 0 | Max wait timeout in milli-second (0: uses same value of default_wait_timeout) |
| --node_cache |  | none | Node cache (none,small,large) |
| --normal_tx_pool |  | 0 | Size of normal transaction pool |
| --patch_tx_pool |  | 0 | Size of patch transaction pool |
| --role |  | 3 | [0:None, 1:Seed, 2:Validator, 3:Both] |
| --secure_aeads |  | chacha,aes128,aes256 | Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string |
| --secure_suites |  | none,tls,ecdhe | Supported Secure suites with order (none,tls,ecdhe) - Comma separated string |
| --seed |  |  | List of trust-seed ip-port, Comma separated string |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --db_type |  |  | Database type(default:original database type) |
| --height |  | 0 | Block Height |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
` goloop chain reset CID `

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --god, -g |  |  | Address or keystore of GOD |
| --validator, -v |  | [] | Address or keystore of Validator, [Validator...] |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c |  | [] | Chain configuration |
| --fee |  | none | Fee configuration (none,icon) |
| --god, -g |  |  | Address or keystore of GOD |
| --out, -o |  | genesis.json | Output file path |
| --supply, -s |  | 0x2961fff8ca4a62327800000 | Total supply of the chain |
| --treasury, -t |  | hx1000000000000000000000000000000000000000 | Treasury address |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --input, -i |  | genesis.json | Input file or directory path |
| --out, -o |  | gs.zip | Output file path |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --cid_only, -c |  | false | Showing chain ID only |
| --nid_only, -n |  | false | Showing network ID only |

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

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --out, -o |  | keystore.json | Output file path |
| --password, -p |  | gochain | Password for the keystore |

### Parent command
|Command | Description|
|---|---|
| [goloop ks](#goloop-ks) |  Keystore manipulation |

### Related commands
|Command | Description|
|---|---|
| [goloop ks gen](#goloop-ks-gen) |  Generate keystore |

## goloop rpc

### Description
JSON-RPC API

### Usage
` goloop rpc `

### Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

### Child commands
|Command | Description|
|---|---|
| [goloop rpc balance](#goloop-rpc-balance) |  GetBalance |
| [goloop rpc blockbyhash](#goloop-rpc-blockbyhash) |  GetBlockByHash |
| [goloop rpc blockbyheight](#goloop-rpc-blockbyheight) |  GetBlockByHeight |
| [goloop rpc blockheaderbyheight](#goloop-rpc-blockheaderbyheight) |  GetBlockHeaderByHeight |
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
` goloop rpc balance ADDRESS `

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --from |  |  | FromAddress |
| --method |  |  | Name of the function to invoke in SCORE, if '--raw' used, will overwrite |
| --param |  | [] | key=value, Function parameters, if '--raw' used, will overwrite |
| --raw |  |  | call with 'data' using raw json file or json-string |
| --to |  |  | ToAddress |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

### Child commands
|Command | Description|
|---|---|
| [goloop rpc monitor block](#goloop-rpc-monitor-block) |  MonitorBlock |
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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --filter |  | [] | EventFilter raw json file or json string |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc monitor block](#goloop-rpc-monitor-block) |  MonitorBlock |
| [goloop rpc monitor event](#goloop-rpc-monitor-event) |  MonitorEvent |

## goloop rpc monitor event

### Description
MonitorEvent

### Usage
` goloop rpc monitor event HEIGHT [flags] `

### Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --addr |  |  | SCORE Address |
| --data |  | [] | Not indexed Arguments of Event, comma-separated string |
| --event |  |  | Signature of Event |
| --indexed |  | [] | Indexed Arguments of Event, comma-separated string |
| --raw |  |  | EventFilter raw json file or json-string |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc monitor block](#goloop-rpc-monitor-block) |  MonitorBlock |
| [goloop rpc monitor event](#goloop-rpc-monitor-event) |  MonitorEvent |

## goloop rpc proofforresult

### Description
GetProofForResult

### Usage
` goloop rpc proofforresult HASH INDEX `

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |
| [goloop rpc totalsupply](#goloop-rpc-totalsupply) |  GetTotalSupply |
| [goloop rpc txbyhash](#goloop-rpc-txbyhash) |  GetTransactionByHash |
| [goloop rpc txresult](#goloop-rpc-txresult) |  GetTransactionResult |
| [goloop rpc votesbyheight](#goloop-rpc-votesbyheight) |  GetVotesByHeight |

## goloop rpc scoreapi

### Description
GetScoreApi

### Usage
` goloop rpc scoreapi ADDRESS `

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --key_password | GOLOOP_RPC_KEY_PASSWORD |  | Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET |  | Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE |  | KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID |  | Network ID |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | 0 | StepLimit |
| --wait | GOLOOP_RPC_WAIT | false | Wait transaction result |
| --wait_interval | GOLOOP_RPC_WAIT_INTERVAL | 1000 | Polling interval(msec) for wait transaction result |
| --wait_timeout | GOLOOP_RPC_WAIT_TIMEOUT | 10 | Timeout(sec) for wait transaction result |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

### Child commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file |
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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --method |  |  | Name of the function to invoke in SCORE, if '--raw' used, will overwrite |
| --param |  | [] | key=value, Function parameters, if '--raw' used, will overwrite |
| --raw |  |  | call with 'data' using raw json file or json-string |
| --to |  |  | ToAddress |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --key_password | GOLOOP_RPC_KEY_PASSWORD |  | Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET |  | Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE |  | KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID |  | Network ID |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | 0 | StepLimit |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |
| --wait | GOLOOP_RPC_WAIT | false | Wait transaction result |
| --wait_interval | GOLOOP_RPC_WAIT_INTERVAL | 1000 | Polling interval(msec) for wait transaction result |
| --wait_timeout | GOLOOP_RPC_WAIT_TIMEOUT | 10 | Timeout(sec) for wait transaction result |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc sendtx deploy

### Description
Deploy Transaction

### Usage
` goloop rpc sendtx deploy SCORE_ZIP_FILE [flags] `

### Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --content_type |  | application/zip | Mime-type of the content |
| --param |  | [] | key=value, Function parameters will be delivered to on_install() or on_update() |
| --to |  | cx0000000000000000000000000000000000000000 | ToAddress |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --key_password | GOLOOP_RPC_KEY_PASSWORD |  | Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET |  | Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE |  | KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID |  | Network ID |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | 0 | StepLimit |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |
| --wait | GOLOOP_RPC_WAIT | false | Wait transaction result |
| --wait_interval | GOLOOP_RPC_WAIT_INTERVAL | 1000 | Polling interval(msec) for wait transaction result |
| --wait_timeout | GOLOOP_RPC_WAIT_TIMEOUT | 10 | Timeout(sec) for wait transaction result |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc sendtx raw

### Description
Send transaction with json file

### Usage
` goloop rpc sendtx raw FILE [flags] `

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --key_password | GOLOOP_RPC_KEY_PASSWORD |  | Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET |  | Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE |  | KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID |  | Network ID |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | 0 | StepLimit |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |
| --wait | GOLOOP_RPC_WAIT | false | Wait transaction result |
| --wait_interval | GOLOOP_RPC_WAIT_INTERVAL | 1000 | Polling interval(msec) for wait transaction result |
| --wait_timeout | GOLOOP_RPC_WAIT_TIMEOUT | 10 | Timeout(sec) for wait transaction result |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc sendtx transfer

### Description
Coin Transfer Transaction

### Usage
` goloop rpc sendtx transfer [flags] `

### Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --message |  |  | Message |
| --to |  |  | ToAddress |
| --value |  |  | Value |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --key_password | GOLOOP_RPC_KEY_PASSWORD |  | Password for the KeyStore file |
| --key_secret | GOLOOP_RPC_KEY_SECRET |  | Secret(password) file for KeyStore |
| --key_store | GOLOOP_RPC_KEY_STORE |  | KeyStore file for wallet |
| --nid | GOLOOP_RPC_NID |  | Network ID |
| --step_limit | GOLOOP_RPC_STEP_LIMIT | 0 | StepLimit |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |
| --wait | GOLOOP_RPC_WAIT | false | Wait transaction result |
| --wait_interval | GOLOOP_RPC_WAIT_INTERVAL | 1000 | Polling interval(msec) for wait transaction result |
| --wait_timeout | GOLOOP_RPC_WAIT_TIMEOUT | 10 | Timeout(sec) for wait transaction result |

### Parent command
|Command | Description|
|---|---|
| [goloop rpc sendtx](#goloop-rpc-sendtx) |  SendTransaction |

### Related commands
|Command | Description|
|---|---|
| [goloop rpc sendtx call](#goloop-rpc-sendtx-call) |  SmartContract Call Transaction |
| [goloop rpc sendtx deploy](#goloop-rpc-sendtx-deploy) |  Deploy Transaction |
| [goloop rpc sendtx raw](#goloop-rpc-sendtx-raw) |  Send transaction with json file |
| [goloop rpc sendtx transfer](#goloop-rpc-sendtx-transfer) |  Coin Transfer Transaction |

## goloop rpc totalsupply

### Description
GetTotalSupply

### Usage
` goloop rpc totalsupply `

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --debug | GOLOOP_RPC_DEBUG | false | JSON-RPC Response with detail information |
| --uri | GOLOOP_RPC_URI |  | URI of JSON-RPC API |

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
| [goloop rpc call](#goloop-rpc-call) |  Call |
| [goloop rpc databyhash](#goloop-rpc-databyhash) |  GetDataByHash |
| [goloop rpc lastblock](#goloop-rpc-lastblock) |  GetLastBlock |
| [goloop rpc monitor](#goloop-rpc-monitor) |  Monitor |
| [goloop rpc proofforresult](#goloop-rpc-proofforresult) |  GetProofForResult |
| [goloop rpc raw](#goloop-rpc-raw) |  Rpc with raw json file |
| [goloop rpc scoreapi](#goloop-rpc-scoreapi) |  GetScoreApi |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --backup_dir | GOLOOP_BACKUP_DIR |  | Node backup directory (default: [node_dir]/backup |
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --console_level | GOLOOP_CONSOLE_LEVEL | trace | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket | GOLOOP_EE_SOCKET |  | Execution engine socket path |
| --engines | GOLOOP_ENGINES | python | Execution engines, comma-separated (python,java) |
| --key_password | GOLOOP_KEY_PASSWORD |  | Password for the KeyStore file |
| --key_secret | GOLOOP_KEY_SECRET |  | Secret (password) file for KeyStore |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --log_forwarder_address | GOLOOP_LOG_FORWARDER_ADDRESS |  | LogForwarder address |
| --log_forwarder_level | GOLOOP_LOG_FORWARDER_LEVEL | info | LogForwarder level |
| --log_forwarder_name | GOLOOP_LOG_FORWARDER_NAME |  | LogForwarder name |
| --log_forwarder_options | GOLOOP_LOG_FORWARDER_OPTIONS | [] | LogForwarder options, comma-separated 'key=value' |
| --log_forwarder_vendor | GOLOOP_LOG_FORWARDER_VENDOR |  | LogForwarder vendor (fluentd,logstash) |
| --log_level | GOLOOP_LOG_LEVEL | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --log_writer_compress | GOLOOP_LOG_WRITER_COMPRESS | false | Use gzip on rotated log file |
| --log_writer_filename | GOLOOP_LOG_WRITER_FILENAME |  | Log filename (rotated files resides in same directory) |
| --log_writer_localtime | GOLOOP_LOG_WRITER_LOCALTIME | false | Use localtime on rotated log file instead of UTC |
| --log_writer_maxage | GOLOOP_LOG_WRITER_MAXAGE | 0 | Maximum age of log file in day |
| --log_writer_maxbackups | GOLOOP_LOG_WRITER_MAXBACKUPS | 0 | Maximum number of backups |
| --log_writer_maxsize | GOLOOP_LOG_WRITER_MAXSIZE | 100 | Maximum log file size in MiB |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory (default: [configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path (default: [node_dir]/cli.sock) |
| --p2p | GOLOOP_P2P | 127.0.0.1:8080 | Advertise ip-port of P2P |
| --p2p_listen | GOLOOP_P2P_LISTEN |  | Listen ip-port of P2P |
| --rpc_addr | GOLOOP_RPC_ADDR | :9080 | Listen ip-port of JSON-RPC |
| --rpc_dump | GOLOOP_RPC_DUMP | false | JSON-RPC Request, Response Dump flag |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --save_key_store |  |  | KeyStore File path to save |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --backup_dir | GOLOOP_BACKUP_DIR |  | Node backup directory (default: [node_dir]/backup |
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --console_level | GOLOOP_CONSOLE_LEVEL | trace | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket | GOLOOP_EE_SOCKET |  | Execution engine socket path |
| --engines | GOLOOP_ENGINES | python | Execution engines, comma-separated (python,java) |
| --key_password | GOLOOP_KEY_PASSWORD |  | Password for the KeyStore file |
| --key_secret | GOLOOP_KEY_SECRET |  | Secret (password) file for KeyStore |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --log_forwarder_address | GOLOOP_LOG_FORWARDER_ADDRESS |  | LogForwarder address |
| --log_forwarder_level | GOLOOP_LOG_FORWARDER_LEVEL | info | LogForwarder level |
| --log_forwarder_name | GOLOOP_LOG_FORWARDER_NAME |  | LogForwarder name |
| --log_forwarder_options | GOLOOP_LOG_FORWARDER_OPTIONS | [] | LogForwarder options, comma-separated 'key=value' |
| --log_forwarder_vendor | GOLOOP_LOG_FORWARDER_VENDOR |  | LogForwarder vendor (fluentd,logstash) |
| --log_level | GOLOOP_LOG_LEVEL | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --log_writer_compress | GOLOOP_LOG_WRITER_COMPRESS | false | Use gzip on rotated log file |
| --log_writer_filename | GOLOOP_LOG_WRITER_FILENAME |  | Log filename (rotated files resides in same directory) |
| --log_writer_localtime | GOLOOP_LOG_WRITER_LOCALTIME | false | Use localtime on rotated log file instead of UTC |
| --log_writer_maxage | GOLOOP_LOG_WRITER_MAXAGE | 0 | Maximum age of log file in day |
| --log_writer_maxbackups | GOLOOP_LOG_WRITER_MAXBACKUPS | 0 | Maximum number of backups |
| --log_writer_maxsize | GOLOOP_LOG_WRITER_MAXSIZE | 100 | Maximum log file size in MiB |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory (default: [configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path (default: [node_dir]/cli.sock) |
| --p2p | GOLOOP_P2P | 127.0.0.1:8080 | Advertise ip-port of P2P |
| --p2p_listen | GOLOOP_P2P_LISTEN |  | Listen ip-port of P2P |
| --rpc_addr | GOLOOP_RPC_ADDR | :9080 | Listen ip-port of JSON-RPC |
| --rpc_dump | GOLOOP_RPC_DUMP | false | JSON-RPC Request, Response Dump flag |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --auth_skip_if_empty_users |  | false | Skip admin API authentication if empty users |
| --cpuprofile |  |  | CPU Profiling data file |
| --memprofile |  |  | Memory Profiling data file |
| --mod_level |  | [] | Set console log level for specific module ('mod'='level',...) |
| --nid_for_p2p |  | false | Use NID instead of CID for p2p network |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --backup_dir | GOLOOP_BACKUP_DIR |  | Node backup directory (default: [node_dir]/backup |
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --console_level | GOLOOP_CONSOLE_LEVEL | trace | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket | GOLOOP_EE_SOCKET |  | Execution engine socket path |
| --engines | GOLOOP_ENGINES | python | Execution engines, comma-separated (python,java) |
| --key_password | GOLOOP_KEY_PASSWORD |  | Password for the KeyStore file |
| --key_secret | GOLOOP_KEY_SECRET |  | Secret (password) file for KeyStore |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --log_forwarder_address | GOLOOP_LOG_FORWARDER_ADDRESS |  | LogForwarder address |
| --log_forwarder_level | GOLOOP_LOG_FORWARDER_LEVEL | info | LogForwarder level |
| --log_forwarder_name | GOLOOP_LOG_FORWARDER_NAME |  | LogForwarder name |
| --log_forwarder_options | GOLOOP_LOG_FORWARDER_OPTIONS | [] | LogForwarder options, comma-separated 'key=value' |
| --log_forwarder_vendor | GOLOOP_LOG_FORWARDER_VENDOR |  | LogForwarder vendor (fluentd,logstash) |
| --log_level | GOLOOP_LOG_LEVEL | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --log_writer_compress | GOLOOP_LOG_WRITER_COMPRESS | false | Use gzip on rotated log file |
| --log_writer_filename | GOLOOP_LOG_WRITER_FILENAME |  | Log filename (rotated files resides in same directory) |
| --log_writer_localtime | GOLOOP_LOG_WRITER_LOCALTIME | false | Use localtime on rotated log file instead of UTC |
| --log_writer_maxage | GOLOOP_LOG_WRITER_MAXAGE | 0 | Maximum age of log file in day |
| --log_writer_maxbackups | GOLOOP_LOG_WRITER_MAXBACKUPS | 0 | Maximum number of backups |
| --log_writer_maxsize | GOLOOP_LOG_WRITER_MAXSIZE | 100 | Maximum log file size in MiB |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory (default: [configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path (default: [node_dir]/cli.sock) |
| --p2p | GOLOOP_P2P | 127.0.0.1:8080 | Advertise ip-port of P2P |
| --p2p_listen | GOLOOP_P2P_LISTEN |  | Listen ip-port of P2P |
| --rpc_addr | GOLOOP_RPC_ADDR | :9080 | Listen ip-port of JSON-RPC |
| --rpc_dump | GOLOOP_RPC_DUMP | false | JSON-RPC Request, Response Dump flag |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --interval | GOLOOP_INTERVAL | 1 | Pull interval |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --no-stream | GOLOOP_NO-STREAM | false | Only pull the first metric-statistics |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) |  Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |
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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c |  |  | Parsing configuration file |
| --key_store |  |  | KeyStore file for wallet |
| --node_dir |  |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --format, -f |  |  | Format the output using the given Go template |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --overwrite |  | false | Overwrite existing chain |

### Inherited Options
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c |  |  | Parsing configuration file |
| --key_store |  |  | KeyStore file for wallet |
| --node_dir |  |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c |  |  | Parsing configuration file |
| --key_store |  |  | KeyStore file for wallet |
| --node_dir |  |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c |  |  | Parsing configuration file |
| --key_store |  |  | KeyStore file for wallet |
| --node_dir |  |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
|Name,shorthand | Environment Variable | Default | Description|
|---|---|---|---|
| --config, -c | GOLOOP_CONFIG |  | Parsing configuration file |
| --key_store | GOLOOP_KEY_STORE |  | KeyStore file for wallet |
| --node_dir | GOLOOP_NODE_DIR |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s | GOLOOP_NODE_SOCK |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
| [goloop gn](#goloop-gn) |  Genesis transaction manipulation |
| [goloop gs](#goloop-gs) |  Genesis storage manipulation |
| [goloop ks](#goloop-ks) |  Keystore manipulation |
| [goloop rpc](#goloop-rpc) |  JSON-RPC API |
| [goloop server](#goloop-server) |  Server management |
| [goloop stats](#goloop-stats) |  Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) |  System info |
| [goloop user](#goloop-user) |  User management |
| [goloop version](#goloop-version) |  Print goloop version |

