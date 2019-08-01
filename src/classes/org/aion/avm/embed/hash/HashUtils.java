package org.aion.avm.embed.hash;

//import org.aion.avm.embed.hash.Blake2b;
import org.bouncycastle.crypto.digests.KeccakDigest;

import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;


public class HashUtils {

    /**
     * Computes the sha256 hash of the given input.
     *
     * @param msg Data for hashing
     * @return Hash
     */
    public static byte[] sha256(byte[] msg) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            digest.update(msg);

            return digest.digest();
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
    }

    /**
     * Computes the blake2b-256 hash of the given input.
     *
     * @param msg Data for hashing
     * @return Hash
     */
    public static byte[] blake2b(byte[] msg) {
//        Blake2b digest = Blake2b.Digest.newInstance(32);
//        digest.update(msg);
//        return digest.digest();
        throw new AssertionError("Not implemented");
    }

    /**
     * Computes the keccak-256 hash of the given input.
     *
     * @param msg Data for hashing
     * @return Hash
     */
    public static byte[] keccak256(byte[] msg) {
        KeccakDigest digest = new KeccakDigest(256);

        digest.update(msg, 0, msg.length);

        byte[] hash = new byte[32];
        digest.doFinal(hash, 0);
        return hash;
    }
}
