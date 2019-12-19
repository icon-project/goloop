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

### 4. Run TransactionExecutorTest

```bash
$ gradle app:exectest:run --args='/tmp/ee.socket'
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

```Java
package example;

import avm.Address;
import avm.Blockchain;

import foundation.icon.ee.tooling.abi.EventLog;
import foundation.icon.ee.tooling.abi.External;
import foundation.icon.ee.tooling.abi.Optional;
import foundation.icon.ee.tooling.abi.Payable;

import java.math.BigInteger;

public class SampleToken
{
    private final String name;
    private final String symbol;
    private final int decimals;
    private final BigInteger totalSupply;

    private SampleToken(String name, String symbol, BigInteger decimals, BigInteger initialSupply) {
        this.name = name;
        this.symbol = symbol;
        this.decimals = decimals.intValue();

        this.totalSupply = initialSupply.multiply(BigInteger.TEN.pow(this.decimals));
        TokenStore.putBalance(Blockchain.getOrigin(), this.totalSupply);
    }

    private static SampleToken token;

    public static void onInstall(String name,
                                 String symbol,
                                 BigInteger decimals,
                                 BigInteger initialSupply) {
        token = new SampleToken(name, symbol, decimals, initialSupply);
    }

    @Payable
    public static void fallback() {
    }

    @External(readonly=true)
    public static String name() {
        return token.name;
    }

    @External(readonly=true)
    public static String symbol() {
        return token.symbol;
    }

    @External(readonly=true)
    public static int decimals() {
        return token.decimals;
    }

    @External(readonly=true)
    public static BigInteger totalSupply() {
        return token.totalSupply;
    }

    @External(readonly=true)
    public static BigInteger balanceOf(Address _owner) {
        return TokenStore.getBalance(_owner);
    }

    @External
    public static void transfer(Address _to, BigInteger _value, @Optional byte[] _data) {
        Address _from = Blockchain.getCaller();
        BigInteger fromBalance = TokenStore.getBalance(_from);
        BigInteger toBalance = TokenStore.getBalance(_to);

        Blockchain.require(_value.compareTo(BigInteger.ZERO) >= 0);
        Blockchain.require(fromBalance.compareTo(_value) >= 0);

        TokenStore.putBalance(_from, fromBalance.subtract(_value));
        TokenStore.putBalance(_to, toBalance.add(_value));

        Transfer(_from, _to, _value, _data);
    }

    @EventLog(indexed=3)
    private static void Transfer(Address _from, Address _to, BigInteger _value, byte[] _data) {}
}
```
