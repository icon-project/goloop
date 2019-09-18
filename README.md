# How to Run

## TransactionExecutorTest

```bash
gradle app:exectest:run --args='/tmp/ee.socket uuid1234'
```

## DAppCompiler

```bash
gradle app:dappcomp:run --args='/some/path/dapp.jar'
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
| SCORE initialize | Override `on_install` method| Define `onInstall:"(...)V"` method |

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
    public static void transfer(Address _to, BigInteger _value) {
        Address _from = Blockchain.getCaller();
        BigInteger fromBalance = TokenStore.getBalance(_from);
        BigInteger toBalance = TokenStore.getBalance(_to);

        Blockchain.require(_value.compareTo(BigInteger.ZERO) >= 0);
        Blockchain.require(fromBalance.compareTo(_value) >= 0);

        TokenStore.putBalance(_from, fromBalance.subtract(_value));
        TokenStore.putBalance(_to, toBalance.add(_value));
        Transfer(_from, _to, _value, "Some data".getBytes());
    }

    @EventLog(indexed=3)
    private static void Transfer(Address from, Address to, BigInteger value, byte[] data) {}
}
```
