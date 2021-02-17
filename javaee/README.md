[![Maven Central](https://maven-badges.herokuapp.com/maven-central/foundation.icon/javaee-api/badge.svg)](https://search.maven.org/search?q=g:foundation.icon%20a:javaee-api)
[![javadoc](http://www.javadoc.io/badge/foundation.icon/javaee-api.svg)](http://www.javadoc.io/doc/foundation.icon/javaee-api)

# Java Execution Environment

## How to Run

**[NOTE]** `$JAVAEE_DIR` indicates the top level directory that contains this repository. You need to replace it with your own directory path.

### 1. Compile SampleToken SCORE

```bash
$ cd $JAVAEE_DIR/samples
$ ./compile.sh sampletoken example.SampleToken
...
The jar has been generated at $JAVAEE_DIR/samples/sampletoken/build/dapp.jar
```

### 2. Optimize the dapp.jar with DAppCompiler

```bash
$ cd $JAVAEE_DIR
$ ./gradlew app:dappcomp:run --args='$JAVAEE_DIR/samples/sampletoken/build/dapp.jar -debug'
...
[main] INFO DAppCompiler - Generated $JAVAEE_DIR/samples/sampletoken/build/optimized-debug.jar
```

### 3. Deploy the optimized jar

If you use `mock_server`, copy the optimized jar into the target directory.
If you use `goloop`, create a deploy transaction with the optimized jar and deploy it.

The counterpart SM server (`mock_server` or `goloop`) should be run first before executing the following step.

### 4. Run Executor Manager

```bash
$ ./gradlew app:execman:run --args='/tmp/ee.socket'
```

## Java SCORE Structure

### Comparison to Python SCORE

| Name | Python | Java |
|------|--------|------|
| External decorator | `@external` | `@External` |
| - (readonly)| `@external(readonly=True)` | `@External(readonly=true)` |
| Payable decorator | `@payable` | `@Payable` |
| Eventlog decorator | `@eventlog` | `@EventLog` |
| - (indexed) | `@eventlog(indexed=1)` | `@EventLog(indexed=1)` |
| fallback signature | `def fallback` | `void fallback()` |
| SCORE initialize | override `on_install` method | define a public constructor |
| Default parameters | native language support | `@Optional` |

**[NOTE]** All external Java methods must have a `public` modifier, and should be instance methods.

### How to invoke a external method of another SCORE

One SCORE can invoke a external method of another SCORE using the following APIs.

```java
// [package score.Context]
public static Object call(Address targetAddress, String method, Object... params);

public static Object call(BigInteger value,
                          Address targetAddress, String method, Object... params);
```

> Example of calling `tokenFallback`
```java
if (_to.isContract()) {
    Context.call(_to, "tokenFallback", _from, _value, dataBytes);
}
```

## Java SCORE Examples

- [SampleToken](./samples/sampletoken/src/example/SampleToken.java)
- [SampleCrowdsale](./samples/crowdsale/src/example/SampleCrowdsale.java)
