package org.aion.avm.embed.crypto;

import org.aion.avm.core.util.Helpers;

import java.security.InvalidKeyException;
import java.security.SignatureException;
import java.security.spec.InvalidKeySpecException;

public class CryptoUtil {
    // Note: In reality this limit will not be reached due to byte array initialization cost (3 nrg per byte).
    // The maximum limit of around 600k for the data was tested. Even though it performs much slower than the libsodium version, it should not cause any problems.
    private static final int VERIFY_EDDSA_MAX_MESSAGE_LENGTH = Integer.MAX_VALUE;

    /**
     * Converts string hex representation to data bytes Accepts following hex: - with or without 0x
     * prefix - with no leading 0, like 0xabc -> 0x0abc
     *
     * @param data String like '0xa5e..' or just 'a5e..'
     * @return decoded bytes array
     */
    public static byte[] hexStringToBytes(String data) {
        if (data == null) {
            return new byte[0];
        }
        if (data.startsWith("0x")) {
            data = data.substring(2);
        }
        if (data.length() % 2 == 1) {
            data = "0" + data;
        }
        return Helpers.hexStringToBytes(data);
    }

    /**
     * Convert a byte-array into a hex String.
     *
     * @param data - byte-array to convert to a hex-string
     * @return hex representation of the data.
     */
    public static String toHexString(byte[] data) {
        return Helpers.bytesToHexString(data);
    }

    /**
     * Sign a byte array of data given the private key.
     *
     * @param data message to be signed
     * @param privateKey byte representation of a private key, length must equal 32
     * @return byte representation of the signature
     * @throws IllegalArgumentException thrown when an input parameter has the wrong size
     */
    public static byte[] signEdDSA(byte[] data, byte[] privateKey){
//        if(VERIFY_EDDSA_MAX_MESSAGE_LENGTH <= data.length){
//            throw new IllegalArgumentException("The input data length exceeds the maximum length allowed of" + VERIFY_EDDSA_MAX_MESSAGE_LENGTH);
//        } else if (privateKey.length != Ed25519Key.SECKEY_BYTES){
//            throw new IllegalArgumentException("Private key length should be equal to " + Ed25519Key.SECKEY_BYTES);
//        }
//
//        try {
//            return Ed25519Key.sign(data, privateKey);
//        } catch (InvalidKeyException | InvalidKeySpecException | SignatureException e) {
//            // This is just used in tests so we don't expect a failure.
//            throw new AssertionError(e);
//        }
          throw new AssertionError("Not implemented");
    }

    /**
     * Verify a message with given signature and public key.
     *
     * @param data message to be verified
     * @param signature signature of the message
     * @param publicKey public key of the message
     * @return result
     * @throws IllegalArgumentException thrown when an input parameter has the wrong size
     */
    public static boolean verifyEdDSA(byte[] data, byte[] signature, byte[] publicKey) {
//        if(VERIFY_EDDSA_MAX_MESSAGE_LENGTH <= data.length){
//            throw new IllegalArgumentException("The input data length exceeds the maximum length allowed of" + VERIFY_EDDSA_MAX_MESSAGE_LENGTH);
//        } else if (signature.length != Ed25519Key.SIG_BYTES){
//            throw new IllegalArgumentException("Signature length should be equal to " + Ed25519Key.SIG_BYTES);
//        } else if (publicKey.length != Ed25519Key.PUBKEY_BYTES){
//            throw new IllegalArgumentException("Public key length should be equal to " + Ed25519Key.PUBKEY_BYTES);
//        }
//
//        try {
//            return Ed25519Key.verify(data, signature, publicKey);
//        } catch (InvalidKeyException | InvalidKeySpecException | SignatureException e) {
//            // This is just used in tests so we don't expect a failure.
//            throw new AssertionError(e);
//        }
        throw new AssertionError("Not implemented");
    }
}
