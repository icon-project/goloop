# Quick Start

## How to Run

**[NOTE]** `$JAVAEE_DIR` indicates the top level directory that contains this repository. You need to replace it with your own directory path.

### 1. Compile SampleToken SCORE

```bash
$ cd $JAVAEE_DIR/samples
$ ./compile.sh sample_token example.SampleToken
...
The jar has been generated at $JAVAEE_DIR/samples/sample_token/build/dapp.jar
```

### 2. Optimize the dapp.jar with DAppCompiler

```bash
$ cd $JAVAEE_DIR
$ gradle app:dappcomp:run --args='$JAVAEE_DIR/samples/sample_token/build/dapp.jar -debug'
...
[main] INFO DAppCompiler - Generated $JAVAEE_DIR/samples/sample_token/build/optimized-debug.jar
```

### 3. Deploy the optimized jar

If you use `mock_server`, copy the optimized jar into the target directory.
If you use `goloop`, create a deploy transaction with the optimized jar and deploy it.

The counterpart SM server (`mock_server` or `goloop`) should be run first before executing the following step.

### 4. Run Executor Manager

```bash
$ gradle app:execman:run --args='/tmp/ee.socket'
```

# Java SCORE Structure

## Comparison to Python SCORE

| Name | Python | Java |
|---|---|---|
| External decorator | @external | @External |
| - (readonly)| @external(readonly=True) | @External(readonly=true) |
| Payable decorator | @payable | @Payable |
| Eventlog decorator | @eventlog | @EventLog |
| - (indexed) | @eventlog(indexed=1) | @EventLog(indexed=1) |
| fallback signature | `def fallback` | `fallback:"()V"` |
| SCORE initialize | override `on_install` method | define `onInstall:"(...)V"` method |
| Default parameters | native language support | `@Optional` |

**[NOTE]** All external Java methods must have `public` and `static` modifiers.

## How to invoke other SCORE's function
[TBD]

## Java SCORE Example

- [SampleToken](./samples/sample_token/src/example/SampleToken.java)
- [SampleCrowdsale](./samples/crowdsale/src/example/SampleCrowdsale.java)
