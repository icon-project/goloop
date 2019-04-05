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
import foundation.icon.icx.transport.jsonrpc.RpcObject;

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
import java.util.concurrent.TimeoutException;

import static foundation.icon.test.common.Env.LOG;

public class Utils {
    public static final BigInteger STATUS_SUCCESS = BigInteger.ONE;
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

    public static Bytes transferIcx(IconService iconService, Wallet fromWallet, Address to, String value) throws IOException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(Env.nodes[0].chains[0].networkId)
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

    // TODO audit on/off에 따른 구현 필요
    public static Bytes deployScore(IconService iconService, Wallet fromWallet, String zipfile, RpcObject params) throws IOException {
        byte[] content = readFile(zipfile);
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(Env.nodes[0].chains[0].networkId)
                .from(fromWallet.getAddress())
                .to(Constants.ZERO_ADDRESS)
                .stepLimit(new BigInteger("300000000"))
                .timestamp(getMicroTime())
                .nonce(new BigInteger("1"))
                .deploy(Constants.CONTENT_TYPE, content)
                .params(params)
                .build();

        SignedTransaction signedTransaction = new SignedTransaction(transaction, fromWallet);
        return iconService.sendTransaction(signedTransaction).execute();
    }

    private static byte[] readFile(String zipfile) throws IOException {
        Path path = Paths.get(zipfile);
        return Files.readAllBytes(path);
    }

    public static TransactionResult getTransactionResult(
            IconService iconService, Bytes txHash, long waitingTime) throws IOException, TimeoutException {
        TransactionResult result = null;
        String msg = null;
        long limitTime = System.currentTimeMillis() + waitingTime;
        while (result == null) {
            try {
                result = iconService.getTransactionResult(txHash).execute();
            } catch (RpcError e) {
                if (limitTime < System.currentTimeMillis()) {
                    throw new TimeoutException();
                }
                try {
                    // wait until block confirmation
                    LOG.debug("RpcError: code(" + e.getCode() + ") message(" + e.getMessage() + "); Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException ie) {
                    ie.printStackTrace();
                }
            }
        }
        LOG.info(msg);
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
}
