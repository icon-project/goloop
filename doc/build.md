# Build guide

## Platform preparation

* GoLang 1.14+

    **Mac OSX**
    ```
    brew install go
    ```
    
* Python 3.7+ Virtual Environment

    **Mac OSX**
    ```
    brew install python
    pip install virtualenv setuptools wheel
    ```

* Rocksdb 6.22+

    **Mac OSX**
    ```
    brew install rocksdb
    ```

## Environment

### Source checkout

First of all, you need to check out the source.
```bash
git clone $REPOSITORY_URL goloop
```

### Prepare virtual environment
```bash
cd $HOME/goloop
virtualenv -p python3 venv
. venv/bin/activate
```

### Install required packages
```bash
pip install -r pyee/requirements.txt
```

## Build

### Build executables

```bash
make
```

Output binaries are placed under `bin/` directory.


### Build python package

```bash
make pyexec
```

Output files are placed under `build/pyee/dist/` directory.

## Quick start

First step, you need to make a configuration for the node.

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

You may send transactions with the wallet, `wallet.json`, for the initial balance of other wallets.

Note that this is a single node configuration.  If you want to make a network with multiple nodes,
you need to make your own genesis and node configurations.
