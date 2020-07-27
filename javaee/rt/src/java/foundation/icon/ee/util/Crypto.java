/*
 * Copyright 2019 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.ee.util;

import org.bouncycastle.asn1.x9.X9ECParameters;
import org.bouncycastle.asn1.x9.X9IntegerConverter;
import org.bouncycastle.crypto.ec.CustomNamedCurves;
import org.bouncycastle.crypto.params.ECDomainParameters;
import org.bouncycastle.math.ec.ECAlgorithms;
import org.bouncycastle.math.ec.ECPoint;
import org.bouncycastle.math.ec.custom.sec.SecP256K1Curve;
import org.bouncycastle.util.BigIntegers;

import java.math.BigInteger;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.Arrays;

public class Crypto {
    public static byte[] sha3_256(byte[] msg) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA3-256");
            return digest.digest(msg);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
    }

    public static byte[] sha256(byte[] msg) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            return digest.digest(msg);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
    }

    public static byte[] recoverKey(byte[] msgHash, byte[] signature, boolean compressed) {
        BigInteger r = BigIntegers.fromUnsignedByteArray(signature, 0, 32);
        BigInteger s = BigIntegers.fromUnsignedByteArray(signature, 32, 32);
        return recoverFromSignature(signature[64], r, s, msgHash, compressed);
    }

    public static byte[] getAddressBytesFromKey(byte[] pubKey) {
        checkArgument(pubKey.length == 33 || pubKey.length == 65, "Invalid key length");
        byte[] uncompressed;
        if (pubKey.length == 33) {
            uncompressed = uncompressKey(pubKey);
        } else {
            uncompressed = pubKey;
        }
        byte[] hash = sha3_256(Arrays.copyOfRange(uncompressed, 1, uncompressed.length));
        byte[] address = new byte[21];
        System.arraycopy(hash, hash.length - 20, address, 1, 20);
        return address;
    }

    private final static X9ECParameters curveParams = CustomNamedCurves.getByName("secp256k1");
    private final static ECDomainParameters curve = new ECDomainParameters(
            curveParams.getCurve(), curveParams.getG(), curveParams.getN(), curveParams.getH());

    private static byte[] uncompressKey(byte[] compKey) {
        ECPoint point = curve.getCurve().decodePoint(compKey);
        byte[] x = point.getXCoord().getEncoded();
        byte[] y = point.getYCoord().getEncoded();
        byte[] key = new byte[x.length + y.length + 1];
        key[0] = 0x04;
        System.arraycopy(x, 0, key, 1, x.length);
        System.arraycopy(y, 0, key, x.length + 1, y.length);
        return key;
    }

    private static ECPoint decompressKey(BigInteger xBN, boolean yBit) {
        X9IntegerConverter x9 = new X9IntegerConverter();
        byte[] compEnc = x9.integerToBytes(xBN, 1 + x9.getByteLength(curve.getCurve()));
        compEnc[0] = (byte) (yBit ? 0x03 : 0x02);
        return curve.getCurve().decodePoint(compEnc);
    }

    private static byte[] recoverFromSignature(int recId, BigInteger r, BigInteger s, byte[] message, boolean compressed) {
        checkArgument(recId >= 0, "recId must be positive");
        checkArgument(r.signum() >= 0, "r must be positive");
        checkArgument(s.signum() >= 0, "s must be positive");

        BigInteger n = curve.getN();
        BigInteger i = BigInteger.valueOf((long) recId / 2);
        BigInteger x = r.add(i.multiply(n));
        BigInteger prime = SecP256K1Curve.q;
        if (x.compareTo(prime) >= 0) {
            return null;
        }
        ECPoint ecPoint = decompressKey(x, (recId & 1) == 1);
        if (!ecPoint.multiply(n).isInfinity()) {
            return null;
        }
        BigInteger e = new BigInteger(1, message);
        BigInteger eInv = BigInteger.ZERO.subtract(e).mod(n);
        BigInteger rInv = r.modInverse(n);
        BigInteger srInv = rInv.multiply(s).mod(n);
        BigInteger eInvrInv = rInv.multiply(eInv).mod(n);
        ECPoint q = ECAlgorithms.sumOfTwoMultiplies(curve.getG(), eInvrInv, ecPoint, srInv);
        return q.getEncoded(compressed);
    }

    private static void checkArgument(boolean expression, String message) {
        if (!expression) {
            throw new IllegalArgumentException(message);
        }
    }
}
