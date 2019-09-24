package org.aion.avm.embed;

import java.math.BigInteger;
import java.nio.ByteBuffer;

import org.aion.types.Transaction;
import org.aion.types.AionAddress;
import org.aion.avm.embed.hash.HashUtils;

/**
 * This is a temporary helper class until the contract address generation logic can be moved
 * into the calling kernel (since it depends on the blockchain design, not the VM).
 */
public class AddressUtil {
    private static final byte CONTRACT_PREFIX = 0x01;

    public static AionAddress generateContractAddress(Transaction tx) {
        long nonceAsLong = new BigInteger(tx.nonce.toByteArray()).longValue();
        ByteBuffer buffer = ByteBuffer.allocate(32 + 8).put(tx.senderAddress.toByteArray()).putLong(nonceAsLong);
        byte[] hash = HashUtils.sha256(buffer.array());
        hash[0] = CONTRACT_PREFIX;
        return new AionAddress(hash);
    }
}
