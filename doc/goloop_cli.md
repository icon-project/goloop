# Goloop

## goloop

### Description
Goloop CLI

### Usage
` goloop [flags] `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |
| [goloop server](#goloop-server) | Server management |
| [goloop stats](#goloop-stats) | Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) | System info |
| [goloop version](#goloop-version) | Print goloop version |

## goloop chain

### Description
Manage chains

### Usage
` goloop chain `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) | Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |
| [goloop server](#goloop-server) | Server management |
| [goloop stats](#goloop-stats) | Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) | System info |
| [goloop version](#goloop-version) | Print goloop version |

## goloop chain import

### Description
Start to import legacy database

### Usage
` goloop chain import NID `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --db_path |  | Database path |
| --height | 0 | Block Height |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

## goloop chain inspect

### Description
Inspect chain

### Usage
` goloop chain inspect NID `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --format, -f |  | Format the output using the given Go template |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

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
| --seed |  | Ip-port of Seed |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

## goloop chain leave

### Description
Leave chain

### Usage
` goloop chain leave NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

## goloop chain ls

### Description
List chains

### Usage
` goloop chain ls `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

## goloop chain reset

### Description
Chain data reset

### Usage
` goloop chain reset NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

## goloop chain start

### Description
Chain start

### Usage
` goloop chain start NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

## goloop chain stop

### Description
Chain stop

### Usage
` goloop chain stop NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

## goloop chain verify

### Description
Chain data verify

### Usage
` goloop chain verify NID `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |

### Related commands
|Command | Description|
|---|---|
| [goloop chain import](#goloop-chain-import) | Start to import legacy database |
| [goloop chain inspect](#goloop-chain-inspect) | Inspect chain |
| [goloop chain join](#goloop-chain-join) | Join chain |
| [goloop chain leave](#goloop-chain-leave) | Leave chain |
| [goloop chain ls](#goloop-chain-ls) | List chains |
| [goloop chain reset](#goloop-chain-reset) | Chain data reset |
| [goloop chain start](#goloop-chain-start) | Chain start |
| [goloop chain stop](#goloop-chain-stop) | Chain stop |
| [goloop chain verify](#goloop-chain-verify) | Chain data verify |

## goloop server

### Description
Server management

### Usage
` goloop server `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --console_level | trace | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --ee_instances | 1 | Number of execution engines |
| --ee_socket |  | Execution engine socket path |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --log_level | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --node_dir |  | Node data directory(default:[configuration file path]/.chain/[ADDRESS]) |
| --p2p | 127.0.0.1:8080 | Advertise ip-port of P2P |
| --p2p_listen |  | Listen ip-port of P2P |
| --rpc_addr | :9080 | Listen ip-port of JSON-RPC |
| --rpc_default_channel |  | JSON-RPC Default Channel |
| --rpc_dump | false | JSON-RPC Request, Response Dump flag |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Child commands
|Command | Description|
|---|---|
| [goloop server save](#goloop-server-save) | Save configuration |
| [goloop server start](#goloop-server-start) | Start server |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) | Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |
| [goloop server](#goloop-server) | Server management |
| [goloop stats](#goloop-stats) | Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) | System info |
| [goloop version](#goloop-version) | Print goloop version |

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
| --cpuprofile |  | CPU Profiling data file |
| --ee_instances | 1 | Number of execution engines |
| --ee_socket |  | Execution engine socket path |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --log_level | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --memprofile |  | Memory Profiling data file |
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
| [goloop server](#goloop-server) | Server management |

### Related commands
|Command | Description|
|---|---|
| [goloop server save](#goloop-server-save) | Save configuration |
| [goloop server start](#goloop-server-start) | Start server |

## goloop server start

### Description
Start server

### Usage
` goloop server start `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --mod_level | [] | Set console log level for specific module (<mod>=<level>,...) |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --console_level | trace | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --cpuprofile |  | CPU Profiling data file |
| --ee_instances | 1 | Number of execution engines |
| --ee_socket |  | Execution engine socket path |
| --key_password |  | Password for the KeyStore file |
| --key_secret |  | Secret(password) file for KeyStore |
| --key_store |  | KeyStore file for wallet |
| --log_level | debug | Global log level (trace,debug,info,warn,error,fatal,panic) |
| --memprofile |  | Memory Profiling data file |
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
| [goloop server](#goloop-server) | Server management |

### Related commands
|Command | Description|
|---|---|
| [goloop server save](#goloop-server-save) | Save configuration |
| [goloop server start](#goloop-server-start) | Start server |

## goloop stats

### Description
Display a live streams of chains metric-statistics

### Usage
` goloop stats `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --interval | 1 | Pull interval |
| --no-stream | false | Only pull the first metric-statistics |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) | Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |
| [goloop server](#goloop-server) | Server management |
| [goloop stats](#goloop-stats) | Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) | System info |
| [goloop version](#goloop-version) | Print goloop version |

## goloop system

### Description
System info

### Usage
` goloop system `

### Options
|Name,shorthand | Default | Description|
|---|---|---|
| --format, -f |  | Format the output using the given Go template |

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) | Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |
| [goloop server](#goloop-server) | Server management |
| [goloop stats](#goloop-stats) | Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) | System info |
| [goloop version](#goloop-version) | Print goloop version |

## goloop version

### Description
Print goloop version

### Usage
` goloop version `

### Inherited Options
|Name,shorthand | Default | Description|
|---|---|---|
| --config, -c |  | Parsing configuration file |
| --cpuprofile |  | CPU Profiling data file |
| --memprofile |  | Memory Profiling data file |
| --node_sock, -s |  | Node Command Line Interface socket path(default:[node_dir]/cli.sock) |

### Parent command
|Command | Description|
|---|---|
| [goloop](#goloop) | Goloop CLI |

### Related commands
|Command | Description|
|---|---|
| [goloop chain](#goloop-chain) | Manage chains |
| [goloop server](#goloop-server) | Server management |
| [goloop stats](#goloop-stats) | Display a live streams of chains metric-statistics |
| [goloop system](#goloop-system) | System info |
| [goloop version](#goloop-version) | Print goloop version |

