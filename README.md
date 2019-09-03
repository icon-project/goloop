# Run TransactionExecutorTest

```bash
make run
```
or

```bash
java -cp ./build/client.jar:$(echo lib/* | tr ' ' ':'):./build \
    -Djava.library.path=./build \
    -Dorg.slf4j.simpleLogger.defaultLogLevel=DEBUG \
    TransactionExecutorTest /tmp/ee.socket uuid1234
```

# Run DAppCompiler

```bash
java -cp ./build/client.jar:$(echo lib/* | tr ' ' ':'):./build \
    -Dorg.slf4j.simpleLogger.defaultLogLevel=DEBUG \
    DAppCompiler ./dapp.jar
```
