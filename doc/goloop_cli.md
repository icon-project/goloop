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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop chain

### Description
Manage chains

### Usage
` goloop chain `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop chain import

### Description
Start to import legacy database

### Usage
` goloop chain import NID [flags] `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --db_path |  | Database path |
| --height | 0 | Block Height |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain inspect

### Description
Inspect chain

### Usage
` goloop chain inspect NID [flags] `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --format, -f |  | Format the output using the given Go template |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
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
|Name,shorthand | Default | Description|
|---|---|---|
| --channel |  | Channel |
| --concurrency | 1 | Maximum number of executors to use for concurrency |
| --db_type | goleveldb | Name of database system(*badgerdb, goleveldb, boltdb, mapdb) |
| --genesis |  | Genesis storage path |
| --genesis_template |  | Genesis template directory or file |
| --max_block_tx_bytes | 0 | Max size of transactions in a block |
| --normal_tx_pool | 0 | Size of normal transaction pool |
| --patch_tx_pool | 0 | Size of patch transaction pool |
| --role | 3 | [0:None, 1:Seed, 2:Validator, 3:Both] |
| --secure_aeads | chacha,aes128,aes256 | Supported Secure AEAD with order (chacha,aes128,aes256) - Comma separated string |
| --secure_suites | none,tls,ecdhe | Supported Secure suites with order (none,tls,ecdhe) - Comma separated string |
| --seed | [] | Ip-port of Seed |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain leave

### Description
Leave chain

### Usage
` goloop chain leave NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
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
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain reset

### Description
Chain data reset

### Usage
` goloop chain reset NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain start

### Description
Chain start

### Usage
` goloop chain start NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain stop

### Description
Chain stop

### Usage
` goloop chain stop NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
| [goloop chain reset](#goloop-chain-reset) |  Chain data reset |
| [goloop chain start](#goloop-chain-start) |  Chain start |
| [goloop chain stop](#goloop-chain-stop) |  Chain stop |
| [goloop chain verify](#goloop-chain-verify) |  Chain data verify |

## goloop chain verify

### Description
Chain data verify

### Usage
` goloop chain verify NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) |  Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) |  Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) |  Inspect chain |
| [goloop chain join](#goloop-chain-join) |  Join chain |
| [goloop chain leave](#goloop-chain-leave) |  Leave chain |
| [goloop chain ls](#goloop-chain-ls) |  List chains |
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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop gn edit

### Description
Edit genesis transaction

### Usage
` goloop gn edit [genesis file] `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --god, -g |  | Address or keystore of GOD |
| --validator, -v | [] | Address or keystore of Validator, [Validator...] |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --god, -g |  | Address or keystore of GOD |
| --out, -o | genesis.json | Output file path |
| --supply, -s | 0x2961fff8ca4a62327800000 | Total supply of the chain |
| --treasury, -t | hx1000000000000000000000000000000000000000 | Treasury address |

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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop gs gen

### Description
Create genesis storage from the template

### Usage
` goloop gs gen `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --input, -i | genesis.json | Input file or directory path |
| --out, -o | gs.zip | Output file path |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --nid_only, -n | false | Showing network ID only |

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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop ks gen

### Description
Generate keystore

### Usage
` goloop ks gen `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --out, -o | keystore.json | Output file path |
| --password, -p | gochain | Password for the keystore |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop rpc balance

### Description
GetBalance

### Usage
` goloop rpc balance ADDRESS `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --data |  | Data (JSON string or file) |
| --data_method |  | Method of Data, will overwrite |
| --data_param | [] | Params of Data, key=value pair, will overwrite |
| --from |  | FromAddress |
| --to |  | ToAddress |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
` goloop rpc monitor block HEIGHT `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --addr |  | Addr |
| --data | [] | Data |
| --event |  | Event |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --nid |  | Network ID, HexString |
| --step_limit | 0 | StepLimit |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --method |  | Name of the function to invoke in SCORE |
| --param | [] | key=value, Function parameters |
| --to |  | ToAddress |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --nid |  | Network ID, HexString |
| --step_limit | 0 | StepLimit |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --content_type | application/zip | Mime-type of the content |
| --param | [] | key=value, Function parameters will be delivered to on_install() or on_update() |
| --to | cx0000000000000000000000000000000000000000 | ToAddress |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --nid |  | Network ID, HexString |
| --step_limit | 0 | StepLimit |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --nid |  | Network ID, HexString |
| --step_limit | 0 | StepLimit |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --message |  | Message |
| --to |  | ToAddress |
| --value | 0 | Value |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --nid |  | Network ID, HexString |
| --step_limit | 0 | StepLimit |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --endpoint |  | Server endpoint |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --console_level | trace | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket |  | Execution engine socket path |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --log_level | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |
| --p2p | 127.0.0.1:8080 | Advertise ip-port of P2P |
| --p2p_listen |  | Listen ip-port of P2P |
| --rpc_addr | :9080 | Listen ip-port of JSON-RPC |
| --rpc_default_channel |  | JSON-RPC Default Channel |
| --rpc_dump | false | JSON-RPC Request, Response Dump flag |

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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop server save

### Description
Save configuration

### Usage
` goloop server save [file] [flags] `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --save_key_store |  | KeyStore File path to save |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --console_level | trace | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket |  | Execution engine socket path |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --log_level | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |
| --p2p | 127.0.0.1:8080 | Advertise ip-port of P2P |
| --p2p_listen |  | Listen ip-port of P2P |
| --rpc_addr | :9080 | Listen ip-port of JSON-RPC |
| --rpc_default_channel |  | JSON-RPC Default Channel |
| --rpc_dump | false | JSON-RPC Request, Response Dump flag |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --mod_level | [] | Set console log level for specific module ('mod'='level',...) |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --console_level | trace | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_socket |  | Execution engine socket path |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --log_level | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |
| --p2p | 127.0.0.1:8080 | Advertise ip-port of P2P |
| --p2p_listen |  | Listen ip-port of P2P |
| --rpc_addr | :9080 | Listen ip-port of JSON-RPC |
| --rpc_default_channel |  | JSON-RPC Default Channel |
| --rpc_dump | false | JSON-RPC Request, Response Dump flag |

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
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --interval | 1 | Pull interval |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --no-stream | false | Only pull the first metric-statistics |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop system

### Description
System info

### Usage
` goloop system `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop system config](#goloop-system-config) |  Configure system |
| [goloop system info](#goloop-system-info) |  Get system information |

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
| [goloop version](#goloop-version) |  Print goloop version |

## goloop system config

### Description
Configure system

### Usage
` goloop system config KEY VALUE `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop system](#goloop-system) |  System info |

### Related commands
|Command | Description|
|---|---|
| [goloop system config](#goloop-system-config) |  Configure system |
| [goloop system info](#goloop-system-info) |  Get system information |

## goloop system info

### Description
Get system information

### Usage
` goloop system info [flags] `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --format, -f |  | Format the output using the given Go template |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop system](#goloop-system) |  System info |

### Related commands
|Command | Description|
|---|---|
| [goloop system config](#goloop-system-config) |  Configure system |
| [goloop system info](#goloop-system-info) |  Get system information |

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
| [goloop version](#goloop-version) |  Print goloop version |

