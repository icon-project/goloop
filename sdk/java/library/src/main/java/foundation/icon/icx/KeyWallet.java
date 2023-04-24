/*
 * Copyright 2018 ICON Foundation
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

package foundation.icon.icx;

import foundation.icon.icx.crypto.ECDSASignature;
import foundation.icon.icx.crypto.IconKeys;
import foundation.icon.icx.crypto.KeyStoreUtils;
import foundation.icon.icx.crypto.Keystore;
import foundation.icon.icx.crypto.KeystoreException;
import foundation.icon.icx.crypto.KeystoreFile;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;

import java.io.File;
import java.io.IOException;
import java.math.BigInteger;
import java.security.InvalidAlgorithmParameterException;
import java.security.NoSuchAlgorithmException;
import java.security.NoSuchProviderException;

import static foundation.icon.icx.TransactionBuilder.checkArgument;

/**
 * An implementation of Wallet which uses of the key pair.
 */
@SuppressWarnings({"WeakerAccess", "unused"})
public class KeyWallet implements Wallet {

    private final Bytes privateKey;
    private final Bytes publicKey;

    private KeyWallet(Bytes privateKey, Bytes publicKey) {
        this.privateKey = privateKey;
        this.publicKey = publicKey;
    }

    /**
     * Loads a key wallet from the private key
     *
     * @param privateKey the private key to load
     * @return KeyWallet
     */
    public static KeyWallet load(Bytes privateKey) {
        Bytes publicKey = IconKeys.getPublicKey(privateKey);
        return new KeyWallet(privateKey, publicKey);
    }

    /**
     * Creates a new KeyWallet with generating a new key pair.
     *
     * @return new KeyWallet
     */
    public static KeyWallet create() throws
            InvalidAlgorithmParameterException, NoSuchAlgorithmException, NoSuchProviderException {
        Bytes privateKey = IconKeys.createPrivateKey();
        Bytes publicKey = IconKeys.getPublicKey(privateKey);
        return new KeyWallet(privateKey, publicKey);
    }

    /**
     * Loads a key wallet from the KeyStore file
     *
     * @param password the password of KeyStore
     * @param file     the KeyStore file
     * @return KeyWallet
     */
    public static KeyWallet load(String password, File file) throws IOException, KeystoreException {
        Bytes privateKey = KeyStoreUtils.loadPrivateKey(password, file);
        Bytes pubicKey = IconKeys.getPublicKey(privateKey);
        return new KeyWallet(privateKey, pubicKey);
    }

    /**
     * Stores the KeyWallet as a KeyStore
     *
     * @param wallet               the wallet to store
     * @param password             the password of KeyStore
     * @param destinationDirectory the KeyStore file is stored at.
     * @return name of the KeyStore file
     */
    public static String store(KeyWallet wallet, String password, File destinationDirectory) throws
            KeystoreException, IOException {
        KeystoreFile keystoreFile = Keystore.create(password, wallet.getPrivateKey(), 1 << 14, 1);
        return KeyStoreUtils.generateWalletFile(keystoreFile, destinationDirectory);
    }

    /**
     * @see Wallet#getAddress()
     */
    @Override
    public Address getAddress() {
        return IconKeys.getAddress(publicKey);
    }

    /**
     * @see Wallet#sign(byte[])
     */
    @Override
    public byte[] sign(byte[] data) {
        checkArgument(data, "hash not found");
        ECDSASignature signature = new ECDSASignature(privateKey);
        BigInteger[] sig = signature.generateSignature(data);
        return signature.recoverableSerialize(sig, data);
    }

    /**
     * Gets the private key
     *
     * @return private key
     */
    public Bytes getPrivateKey() {
        return privateKey;
    }

    /**
     * Gets the public key
     *
     * @return public key in uncompressed format
     */
    public Bytes getPublicKey() {
        return publicKey;
    }

    /**
     * Gets the public key in the specified format
     *
     * @param compressed whether to generate a compressed format
     * @return public key in the specified format
     */
    public Bytes getPublicKey(boolean compressed) {
        return IconKeys.getPublicKey(privateKey, compressed);
    }
}
