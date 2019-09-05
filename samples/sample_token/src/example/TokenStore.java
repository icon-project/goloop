package example;

import avm.Address;
import avm.Blockchain;
import org.aion.avm.userlib.AionBuffer;

import java.math.BigInteger;

class TokenStore {
    protected enum Prefix {
        BALANCE_MAP,
    }

    static void putBalance(Address owner, BigInteger amount) {
        Blockchain.putStorage(encodeKey(Prefix.BALANCE_MAP, owner), amount.toByteArray());
    }

    static BigInteger getBalance(Address owner) {
        byte[] result = Blockchain.getStorage(encodeKey(Prefix.BALANCE_MAP, owner));
        return result != null ? new BigInteger(result) : BigInteger.ZERO;
    }

    private static byte[] encodeKey(Prefix prefix, Address address) {
        return Blockchain.keccak256(
                AionBuffer.allocate(Integer.BYTES + Address.LENGTH)
                          .putInt(prefix.hashCode())
                          .putAddress(address)
                          .getArray());
    }
}
