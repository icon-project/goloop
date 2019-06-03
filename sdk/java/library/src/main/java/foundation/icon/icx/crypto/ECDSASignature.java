package foundation.icon.icx.crypto;

import foundation.icon.icx.data.Bytes;
import org.bouncycastle.asn1.x9.X9ECParameters;
import org.bouncycastle.asn1.x9.X9IntegerConverter;
import org.bouncycastle.crypto.digests.SHA256Digest;
import org.bouncycastle.crypto.ec.CustomNamedCurves;
import org.bouncycastle.crypto.params.ECDomainParameters;
import org.bouncycastle.crypto.params.ECPrivateKeyParameters;
import org.bouncycastle.crypto.signers.ECDSASigner;
import org.bouncycastle.crypto.signers.HMacDSAKCalculator;
import org.bouncycastle.math.ec.ECAlgorithms;
import org.bouncycastle.math.ec.ECPoint;
import org.bouncycastle.math.ec.custom.sec.SecP256K1Curve;
import org.bouncycastle.util.BigIntegers;

import java.math.BigInteger;
import java.nio.ByteBuffer;
import java.util.Arrays;

/**
 * Original Code
 * https://github.com/web3j/web3j/blob/master/crypto/src/main/java/org/web3j/crypto/Sign.java
 * <p>Transaction signing logic.</p>
 *
 * <p>Adapted from the
 * <a href="https://github.com/bitcoinj/bitcoinj/blob/master/core/src/main/java/org/bitcoinj/core/ECKey.java">
 * BitcoinJ ECKey</a> implementation.
 */
public class ECDSASignature {

    private final X9ECParameters curveParams = CustomNamedCurves.getByName("secp256k1");
    private final ECDomainParameters curve = new ECDomainParameters(
            curveParams.getCurve(), curveParams.getG(), curveParams.getN(), curveParams.getH());

    private Bytes privateKey;

    public ECDSASignature(Bytes privateKey) {
        this.privateKey = privateKey;
    }

    /**
     * Serialize the result of `generateSignature` and recovery id
     *
     * @param sig     the R and S components of the signature, wrapped.
     * @param message Hash of the data that was signed.
     * @return 32 bytes for R + 32 bytes for S + 1 byte for recovery id
     */
    public byte[] recoverableSerialize(BigInteger[] sig, byte[] message) {
        byte recId = findRecoveryId(sig, message);

        ByteBuffer buffer = ByteBuffer.allocate(32 + 32 + 1);
        buffer.put(BigIntegers.asUnsignedByteArray(32, sig[0]));
        buffer.put(BigIntegers.asUnsignedByteArray(32, sig[1]));
        buffer.put(recId);
        return buffer.array();
    }

    /**
     * generate a signature for the given message using the key we were initialised with
     *
     * @param message Hash of the data that was signed.
     * @return the R and S components of the signature, wrapped.
     */
    public BigInteger[] generateSignature(byte[] message) {
        BigInteger p = new BigInteger(1, privateKey.toByteArray());
        ECDSASigner signer = new ECDSASigner(new HMacDSAKCalculator(new SHA256Digest()));
        ECPrivateKeyParameters param = new ECPrivateKeyParameters(p, curve);
        signer.init(true, param);
        BigInteger[] sig = signer.generateSignature(message);
        BigInteger r = sig[0];
        BigInteger s = sig[1];
        if (s.compareTo(curveParams.getN().shiftRight(1)) > 0) {
            s = curve.getN().subtract(s);
        }
        return new BigInteger[]{r, s};
    }

    /**
     * Returns the recovery ID, a byte with value between 0 and 3, inclusive, that specifies which of 4 possible
     * curve points was used to sign a message. This value is also referred to as "v".
     *
     * @param sig     the R and S components of the signature, wrapped.
     * @param message Hash of the data that was signed.
     * @return recId   Which possible key to recover.
     * @throws RuntimeException if no recovery ID can be found.
     */
    public byte findRecoveryId(BigInteger[] sig, byte[] message) {
        Bytes publicKey = IconKeys.getPublicKey(privateKey);
        BigInteger p = new BigInteger(1, publicKey.toByteArray());
        byte recId = -1;
        for (byte i = 0; i < 4; i++) {
            BigInteger k = recoverFromSignature(i, sig, message);
            if (k != null && k.equals(p)) {
                recId = i;
                break;
            }
        }
        if (recId == -1) {
            throw new RuntimeException(
                    "Could not construct a recoverable key. This should never happen.");
        }
        return recId;
    }

    /**
     * <p>Given the components of a signature and a selector value, recover and return the public
     * key that generated the signature according to the algorithm in SEC1v2 section 4.1.6.</p>
     *
     * <p>The recId is an index from 0 to 3 which indicates which of the 4 possible keys is the
     * correct one. Because the key recovery operation yields multiple potential keys, the correct
     * key must either be stored alongside the
     * signature, or you must be willing to try each recId in turn until you find one that outputs
     * the key you are expecting.</p>
     *
     * <p>If this method returns null it means recovery was not possible and recId should be
     * iterated.</p>
     *
     * <p>Given the above two points, a correct usage of this method is inside a for loop from
     * 0 to 3, and if the output is null OR a key that is not the one you expect, you try again
     * with the next recId.</p>
     *
     * @param recId   Which possible key to recover.
     * @param sig     the R and S components of the signature, wrapped.
     * @param message Hash of the data that was signed.
     * @return An ECKey containing only the public part, or null if recovery wasn't possible.
     */
    private BigInteger recoverFromSignature(int recId, BigInteger[] sig, byte[] message) {
        BigInteger r = sig[0];
        BigInteger s = sig[1];

        checkArgument(recId >= 0, "recId must be positive");
        checkArgument(r.signum() >= 0, "r must be positive");
        checkArgument(s.signum() >= 0, "s must be positive");
        checkArgument(message != null, "message cannot be null");

        // 1.0 For j from 0 to h   (h == recId here and the loop is outside this function)
        //   1.1 Let x = r + jn
        BigInteger n = curve.getN();  // Curve order.
        BigInteger i = BigInteger.valueOf((long) recId / 2);
        BigInteger x = r.add(i.multiply(n));
        //   1.2. Convert the integer x to an octet string X of length mlen using the conversion
        //        routine specified in Section 2.3.7, where mlen = ⌈(log2 p)/8⌉ or mlen = ⌈m/8⌉.
        //   1.3. Convert the octet string (16 set binary digits)||X to an elliptic curve point R
        //        using the conversion routine specified in Section 2.3.4. If this conversion
        //        routine outputs "invalid", then do another iteration of Step 1.
        //
        // More concisely, what these points mean is to use X as a compressed public key.
        BigInteger prime = SecP256K1Curve.q;
        if (x.compareTo(prime) >= 0) {
            // Cannot have point co-ordinates larger than this as everything takes place modulo Q.
            return null;
        }
        // Compressed keys require you to know an extra bit of data about the y-coord as there are
        // two possibilities. So it's encoded in the recId.
        ECPoint ecPoint = decompressKey(x, (recId & 1) == 1);
        //   1.4. If nR != point at infinity, then do another iteration of Step 1 (callers
        //        responsibility).
        if (!ecPoint.multiply(n).isInfinity()) {
            return null;
        }
        //   1.5. Compute e from M using Steps 2 and 3 of ECDSASignature signature verification.
        BigInteger e = new BigInteger(1, message);
        //   1.6. For k from 1 to 2 do the following.   (loop is outside this function via
        //        iterating recId)
        //   1.6.1. Compute a candidate public key as:
        //               Q = mi(r) * (sR - eG)
        //
        // Where mi(x) is the modular multiplicative inverse. We transform this into the following:
        //               Q = (mi(r) * s ** R) + (mi(r) * -e ** G)
        // Where -e is the modular additive inverse of e, that is z such that z + e = 0 (mod n).
        // In the above equation ** is point multiplication and + is point addition (the EC group
        // operator).
        //
        // We can find the additive inverse by subtracting e from zero then taking the mod. For
        // example the additive inverse of 3 modulo 11 is 8 because 3 + 8 mod 11 = 0, and
        // -3 mod 11 = 8.
        BigInteger eInv = BigInteger.ZERO.subtract(e).mod(n);
        BigInteger rInv = r.modInverse(n);
        BigInteger srInv = rInv.multiply(s).mod(n);
        BigInteger eInvrInv = rInv.multiply(eInv).mod(n);
        ECPoint q = ECAlgorithms.sumOfTwoMultiplies(curve.getG(), eInvrInv, ecPoint, srInv);
        return new BigInteger(1, q.getEncoded(false));
    }

    /**
     * Decompress a compressed public key (x co-ord and low-bit of y-coord).
     */
    private ECPoint decompressKey(BigInteger xBN, boolean yBit) {
        X9IntegerConverter x9 = new X9IntegerConverter();
        byte[] compEnc = x9.integerToBytes(xBN, 1 + x9.getByteLength(curve.getCurve()));
        compEnc[0] = (byte) (yBit ? 0x03 : 0x02);
        return curve.getCurve().decodePoint(compEnc);
    }

    private void checkArgument(boolean expression, String message) {
        if (!expression) {
            throw new IllegalArgumentException(message);
        }
    }
}
