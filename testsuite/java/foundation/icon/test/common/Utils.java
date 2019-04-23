/*
 * Copyright (c) 2019 ICON Foundation
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

package foundation.icon.test.common;

import foundation.icon.icx.*;
import foundation.icon.icx.crypto.KeystoreException;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.data.TransactionResult.EventLog;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.score.GovScore;

import java.io.File;
import java.io.IOException;
import java.math.BigDecimal;
import java.math.BigInteger;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.InvalidAlgorithmParameterException;
import java.security.NoSuchAlgorithmException;
import java.security.NoSuchProviderException;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;

public class Utils {
    public static BigInteger getMicroTime() {
        long timestamp = System.currentTimeMillis() * 1000L;
        return new BigInteger(Long.toString(timestamp));
    }

    public static KeyWallet createAndStoreWallet() throws IOException {
        try {
            KeyWallet wallet = KeyWallet.create();
            KeyWallet.store(wallet, "P@sswOrd", new File("/tmp"));
            return wallet;
        } catch (InvalidAlgorithmParameterException | NoSuchAlgorithmException | NoSuchProviderException e) {
            e.printStackTrace();
            throw new IOException("Key creation failed!");
        } catch (KeystoreException e) {
            e.printStackTrace();
            throw new IOException("Key store failed!");
        }
    }

    public static KeyWallet readWalletFromFile(String path, String password) throws IOException {
        try {
            File file = new File(path);
            return KeyWallet.load(password, file);
        } catch (KeystoreException e) {
            e.printStackTrace();
            throw new IOException("Key load failed!");
        }
    }

    public static void ensureIcxBalance(IconService iconService, Address address, long oldVal, long newVal) throws IOException {
        BigInteger oldValInt = BigInteger.valueOf(oldVal).multiply(BigDecimal.TEN.pow(18).toBigInteger());
        BigInteger newValInt = BigInteger.valueOf(newVal).multiply(BigDecimal.TEN.pow(18).toBigInteger());
        while (true) {
            BigInteger icxBalance = iconService.getBalance(address).execute();
            String msg = "ICX balance of " + address + ": " + icxBalance;
            if (icxBalance.equals(oldValInt)) {
                try {
                    // wait until block confirmation
                    LOG.debug(msg + "; Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
            } else if (icxBalance.equals(newValInt)) {
                LOG.info(msg);
                break;
            } else {
                throw new IOException("ICX balance mismatch!");
            }
        }
    }

    public static Bytes transfer(IconService iconService, BigInteger networkId, Wallet fromWallet, Address to, long value) throws IOException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(networkId)
                .from(fromWallet.getAddress())
                .to(to)
                .value(BigInteger.valueOf(value))
                .stepLimit(new BigInteger("1000"))
                .timestamp(getMicroTime())
                .nonce(new BigInteger("1"))
                .build();

        SignedTransaction signedTransaction = new SignedTransaction(transaction, fromWallet);
        return iconService.sendTransaction(signedTransaction).execute();
    }

    public static Bytes transferIcx(IconService iconService, BigInteger networkId, Wallet fromWallet, Address to, String value) throws IOException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(networkId)
                .from(fromWallet.getAddress())
                .to(to)
                .value(IconAmount.of(value, IconAmount.Unit.ICX).toLoop())
                .stepLimit(new BigInteger("2000000"))
                .timestamp(getMicroTime())
                .nonce(new BigInteger("1"))
                .build();

        SignedTransaction signedTransaction = new SignedTransaction(transaction, fromWallet);
        return iconService.sendTransaction(signedTransaction).execute();
    }

    public static Bytes deployScore(IconService iconService, BigInteger networkId, Wallet fromWallet, Address to, String zipfile, RpcObject params, long stepLimit) throws IOException {
        byte[] content = readFile(zipfile);
        if(stepLimit == -1) {
            stepLimit = 200000;
        }
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(networkId)
                .from(fromWallet.getAddress())
                .to(to)
                .stepLimit(BigInteger.valueOf(stepLimit))
                .timestamp(getMicroTime())
                .nonce(new BigInteger("1"))
                .deploy(Constants.CONTENT_TYPE, content)
                .params(params)
                .build();

        SignedTransaction signedTransaction = new SignedTransaction(transaction, fromWallet);
        return iconService.sendTransaction(signedTransaction).execute();
    }

    public static Bytes installScore(IconService iconService, Env.Chain chain, Wallet fromWallet, String zipfile, RpcObject params, long stepLimit) throws IOException {
        Bytes txHash = deployScore(iconService, chain.networkId, fromWallet, Constants.CHAINSCORE_ADDRESS, zipfile, params, stepLimit);
        if(chain.isAudit()) {
            Bytes acceptHash = new GovScore(iconService, chain).acceptScore(txHash);
            return acceptHash;
        }
        return txHash;
    }

    public static Bytes updateScore(IconService iconService, Env.Chain chain, Wallet fromWallet, Address scoreAddr, String zipfile, RpcObject params, long stepLimit) throws IOException {
        Bytes txHash = deployScore(iconService, chain.networkId, fromWallet, scoreAddr, zipfile, params, stepLimit);
        if(chain.isAudit()) {
            Bytes acceptHash = new GovScore(iconService, chain).acceptScore(txHash);
            return acceptHash;
        }
        return txHash;
    }

    private static byte[] readFile(String zipfile) throws IOException {
        Path path = Paths.get(zipfile);
        return Files.readAllBytes(path);
    }

    public static TransactionResult getTransactionResult(IconService iconService, Bytes txHash, long waitingTime)
            throws ResultTimeoutException, IOException  {
        TransactionResult result = null;
        long limitTime = System.currentTimeMillis() + waitingTime;
        while (result == null) {
            try {
                result = iconService.getTransactionResult(txHash).execute();
            } catch (RpcError e) {
                if (limitTime < System.currentTimeMillis()) {
                    throw new ResultTimeoutException(txHash);
                }
                try {
                    // wait until block confirmation
                    LOG.debug("RpcError: code(" + e.getCode() + ") message(" + e.getMessage() + "); Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException ex) {
                    ex.printStackTrace();
                }
            }
        }
        return result;
    }

    public static EventLog findEventLogWithFuncSig(TransactionResult result, Address scoreAddress, String funcSig) {
        List<EventLog> eventLogs = result.getEventLogs();
        for (EventLog event : eventLogs) {
            if (event.getScoreAddress().equals(scoreAddress.toString())) {
                String signature = event.getIndexed().get(0).asString();
                LOG.debug("function sig: " + signature);
                if (funcSig.equals(signature)) {
                    return event;
                }
            }
        }
        return null;
    }

    public static RpcItem icxCall(
            IconService iconService, BigInteger nid, Wallet fromWallet, Address scoreAddr, String function,
            RpcObject params) throws Exception {
        // TODO NID
        Call.Builder builder = new Call.Builder()
                .from(fromWallet.getAddress())
                .to(scoreAddr)
                .method(function);
        if(params != null) {
            builder.params(params);
        }
        Call<RpcItem> call = builder.build();
        return iconService.call(call).execute();
    }

    public static TransactionResult sendTransactionWithCall(
            IconService iconService, BigInteger nid, Wallet fromWallet, Address scoreAddr, String function,
            RpcObject params, long value) throws ResultTimeoutException, IOException {

        long timestamp = System.currentTimeMillis() * 1000L;
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(nid)
                .from(fromWallet.getAddress())
                .to(scoreAddr)
                .stepLimit(new BigInteger("200000"))
                .timestamp(new BigInteger(Long.toString(timestamp)))
                .nonce(new BigInteger("1"))
                .value(BigInteger.valueOf(value))
                .call(function)
                .params(params)
                .build();

        SignedTransaction signedTransaction = new SignedTransaction(transaction, fromWallet);
        Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        return result;
    }

    public static Bytes sendTransactionWithCall(
            IconService iconService, BigInteger nid, Wallet fromWallet, Address scoreAddr, String function,
            RpcObject params, long value, boolean waitResult) throws TransactionFailureException, ResultTimeoutException, IOException {

        long timestamp = System.currentTimeMillis() * 1000L;
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(nid)
                .from(fromWallet.getAddress())
                .to(scoreAddr)
                .stepLimit(new BigInteger("200000"))
                .timestamp(new BigInteger(Long.toString(timestamp)))
                .nonce(new BigInteger("1"))
                .value(BigInteger.valueOf(value))
                .call(function)
                .params(params)
                .build();

        SignedTransaction signedTransaction = new SignedTransaction(transaction, fromWallet);
        Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
        if(waitResult) {
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new TransactionFailureException(result.getFailure());
            }
        }
        return txHash;
    }
}
