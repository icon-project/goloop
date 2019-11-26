package example;

import avm.Address;
import avm.Blockchain;
import foundation.icon.ee.tooling.abi.External;

import java.math.BigInteger;

public class APITest
{
    public static void onInstall() {
    }

    @External
    public static void getAddress(Address addr) {
        Blockchain.require(Blockchain.getAddress().equals(addr));
    }

    @External(readonly=true)
    public static Address getAddressQuery() {
        return Blockchain.getAddress();
    }

    @External
    public static void getCaller(Address caller) {
        Blockchain.require(Blockchain.getCaller().equals(caller));
    }

    @External(readonly=true)
    public static Address getCallerQuery() {
        return Blockchain.getCaller();
    }

    @External
    public static void getOrigin(Address origin) {
        Blockchain.require(Blockchain.getOrigin().equals(origin));
    }

    @External(readonly=true)
    public static Address getOriginQuery() {
        return Blockchain.getOrigin();
    }

    @External
    public static void getOwner(Address owner) {
        Blockchain.require(Blockchain.getOwner().equals(owner));
    }

    @External(readonly=true)
    public static Address getOwnerQuery() {
        return Blockchain.getOwner();
    }

    @External
    public static BigInteger getValue() {
        return Blockchain.getValue();
    }

    @External(readonly=true)
    public static BigInteger getValueQuery() {
        return Blockchain.getValue();
    }

    @External
    public static void getBlockTimestamp() {
        Blockchain.require(Blockchain.getBlockTimestamp() > 0L);
    }

    @External(readonly=true)
    public static long getBlockTimestampQuery() {
        return Blockchain.getBlockTimestamp();
    }

    @External
    public static void getBlockHeight() {
        Blockchain.require(Blockchain.getBlockHeight() > 0L);
    }

    @External(readonly=true)
    public static long getBlockHeightQuery() {
        return Blockchain.getBlockHeight();
    }
}
