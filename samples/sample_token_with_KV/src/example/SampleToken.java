package example;

import avm.Address;
import avm.Blockchain;
import org.aion.avm.tooling.abi.Callable;

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

        //*** NOTE: #pow() is not implemented in the shadow BigInteger
        //#ORIG
        //this.totalSupply = initialSupply.multiply(BigInteger.TEN.pow(this.decimals));
        //#TEMP
        BigInteger exp = BigInteger.ONE;
        for (int i = 0; i < this.decimals; i++) {
            exp = exp.multiply(BigInteger.TEN);
        }
        this.totalSupply = initialSupply.multiply(exp);
        //#

        TokenStorage.putOwnerBalance(Blockchain.getOrigin(), this.totalSupply);
    }

    private static SampleToken token;

    @Callable
    public static void onInstall(String name, String symbol, BigInteger decimals, BigInteger initialSupply) {
        token = new SampleToken(name, symbol, decimals, initialSupply);
    }

    @Callable
    public static String name() {
        return token.name;
    }

    @Callable
    public static String symbol() {
        return token.symbol;
    }

    @Callable
    public static int decimals() {
        return token.decimals;
    }

    @Callable
    public static BigInteger totalSupply() {
        return token.totalSupply;
    }

    @Callable
    public static BigInteger balanceOf(Address _owner) {
        return TokenStorage.getOwnerBalance(_owner);
    }

    @Callable
    public static void transfer(Address _to, BigInteger _value) {
        Address _from = Blockchain.getCaller();
        BigInteger fromBalance = TokenStorage.getOwnerBalance(_from);
        BigInteger toBalance = TokenStorage.getOwnerBalance(_to);

        Blockchain.require(_value.compareTo(BigInteger.ZERO) >= 0);
        Blockchain.require(fromBalance.compareTo(_value) >= 0);

        TokenStorage.putOwnerBalance(_from, fromBalance.subtract(_value));
        TokenStorage.putOwnerBalance(_to, toBalance.add(_value));
        Blockchain.log("Transfer".getBytes(), _from.toByteArray(), _to.toByteArray(), _value.toByteArray());
    }
}
