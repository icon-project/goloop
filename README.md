# Run Proxy test

```
make run
```
or
```
java -cp build/client.jar:lib/msgpack-core-0.8.17.jar:lib/bcprov-jdk15on-1.60.jar:build -Djava.library.path=build ProxyTest /tmp/ee.socket uuid1234
```
