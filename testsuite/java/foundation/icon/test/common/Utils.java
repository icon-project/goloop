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

import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.IOException;
import java.math.BigDecimal;
import java.math.BigInteger;
import java.nio.file.Files;
import java.util.List;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

public class Utils {
    public static BigInteger getMicroTime() {
        long timestamp = System.currentTimeMillis() * 1000L;
        return new BigInteger(Long.toString(timestamp));
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

    public static void ensureIcxBalance(IconService iconService, Address address, long oldVal, long newVal) throws Exception {
        BigInteger oldValInt = BigInteger.valueOf(oldVal).multiply(BigDecimal.TEN.pow(18).toBigInteger());
        BigInteger newValInt = BigInteger.valueOf(newVal).multiply(BigDecimal.TEN.pow(18).toBigInteger());
        long limitTime = System.currentTimeMillis() + Constants.DEFAULT_WAITING_TIME;
        while (true) {
            BigInteger icxBalance = iconService.getBalance(address).execute();
            String msg = "ICX balance of " + address + ": " + icxBalance;
            if (icxBalance.equals(oldValInt)) {
                if (limitTime < System.currentTimeMillis()) {
                    throw new ResultTimeoutException(null);
                }
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

    public static Bytes transfer(IconService iconService, int networkId, Wallet fromWallet, Address to, BigInteger value, String msg) throws IOException {
        TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                .nid(BigInteger.valueOf(networkId))
                .from(fromWallet.getAddress())
                .to(to)
                .value(value)
                .stepLimit(BigInteger.valueOf(Constants.DEFAULT_STEP_LIMIT))
                .timestamp(getMicroTime())
                .nonce(new BigInteger("1"));
        Transaction transaction;
        if(msg != null) {
            transaction = builder.message(msg).build();
        }
        else {
            transaction = builder.build();
        }

        SignedTransaction signedTransaction = new SignedTransaction(transaction, fromWallet);
        return iconService.sendTransaction(signedTransaction).execute();
    }

    public static Bytes transfer(IconService iconService, int networkId, Wallet fromWallet, Address to, BigInteger value) throws IOException {
        return transfer(iconService, networkId, fromWallet, to, value, null);
    }

    public static Bytes transfer(IconService iconService, int networkId, Wallet fromWallet, Address to, long value) throws IOException {
        return transfer(iconService, networkId, fromWallet, to, BigInteger.valueOf(value), null);
    }

    public static Bytes transferIcx(IconService iconService, int networkId,
            Wallet fromWallet, Address to, String value) throws IOException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(BigInteger.valueOf(networkId))
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

    public static Bytes deployScore(IconService iconService, int networkId,
                                    Wallet fromWallet, Address to, String contentPath, RpcObject params)
            throws IOException {
        return deployScore(iconService, networkId, fromWallet, to, contentPath, params, Constants.DEFAULT_STEP_LIMIT);
    }

    public static Bytes deployScore(IconService iconService, int networkId,
            Wallet fromWallet, Address to, String contentPath, RpcObject params, long stepLimit)
                throws IOException {
        byte[] content = zipContent(contentPath);
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(BigInteger.valueOf(networkId))
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
        return null ;
    }

    public static RpcItem icxCall(
            IconService iconService, Address scoreAddr, String function,
            RpcObject params) throws Exception {
        Call.Builder builder = new Call.Builder()
                .to(scoreAddr)
                .method(function);
        if(params != null) {
            builder.params(params);
        }
        Call<RpcItem> call = builder.build();
        return iconService.call(call).execute();
    }

    public static TransactionResult sendTransactionWithCall(
            IconService iconService, int nid, Wallet fromWallet, Address scoreAddr, String function,
                RpcObject params) throws ResultTimeoutException, IOException {
        return sendTransactionWithCall(iconService, nid, fromWallet, scoreAddr, function, params, 0);
    }

    public static TransactionResult sendTransactionWithCall(
            IconService iconService, int nid, Wallet fromWallet, Address scoreAddr, String function,
            RpcObject params, long value) throws ResultTimeoutException, IOException {

        long timestamp = System.currentTimeMillis() * 1000L;
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(BigInteger.valueOf(nid))
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
        return getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
    }

    public static Bytes sendTransactionWithCall(IconService iconService, int nid, Wallet fromWallet,
            Address scoreAddr, String function, RpcObject params, long value, boolean waitResult)
                    throws TransactionFailureException, ResultTimeoutException, IOException {
        long timestamp = System.currentTimeMillis() * 1000L;
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(BigInteger.valueOf(nid))
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
            TransactionResult result = getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new TransactionFailureException(result.getFailure());
            }
        }
        return txHash;
    }

    public static boolean isAudit(IconService iconService) {
        BigInteger rpcObject;
        try {
            rpcObject = icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getServiceConfig", null).asInteger();
        }
        catch (Exception ex) {
            throw new IllegalStateException("FAIL to call icx for serviceConfig. ex : " + ex.getMessage());
        }
        long lAudit = rpcObject.longValue();
        return (lAudit & 0x2) == 0x2;
    }

    public static TransactionResult acceptIfAuditEnabled(IconService iconService, Env.Chain chain, Bytes txHash) throws ResultTimeoutException, IOException, TransactionFailureException {
        TransactionResult result = null;
        if(Utils.isAudit(iconService)) {
            LOG.infoEntering("accept", "accept score");
            result = new GovScore(iconService, chain).acceptScore(txHash);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                LOG.infoExiting();
                throw new TransactionFailureException(result.getFailure());
            }
            LOG.infoExiting();
        }
        return result;
    }

    private static void recursiveZip(File source, String zipPath, ZipOutputStream zos) throws IOException{
        if(source.isHidden()) {
            return;
        }
        if(source.isDirectory()) {
            String dir = source.getName();
            if(!dir.endsWith(File.separator)) {
                dir = dir + File.separator;
            }
            zos.putNextEntry(new ZipEntry(dir));
            zos.closeEntry();
            File []files = source.listFiles();
            if(files == null) {
                return;
            }
            String path = zipPath == null ? dir : zipPath + dir;
            for(File file : files) {
                recursiveZip(file, path, zos);
            }
        }
        else {
            ZipEntry ze = new ZipEntry(zipPath + source.getName());
            zos.putNextEntry(ze);
            zos.write(Files.readAllBytes(source.toPath()));
            zos.closeEntry();
        }
    }

    public static byte[] zipContent(String path) throws IOException {
        ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
        ZipOutputStream zos = new ZipOutputStream(outputStream);
        recursiveZip(new File(path), null, zos);
        zos.close();
        outputStream.close();
        return outputStream.toByteArray();
    }

    public static void transferAndCheck(IconService service, Env.Chain chain,
                                        KeyWallet from, Address to, BigInteger val) throws Exception{
        BigInteger prevFromBal = service.getBalance(from.getAddress()).execute();
        BigInteger expectedFromBal = prevFromBal.subtract(val);
        assertTrue(expectedFromBal.signum() >= 0);

        BigInteger prevToBal = service.getBalance(to).execute();
        Bytes txHash = Utils.transfer(service, chain.networkId, from, to, val);
        TransactionResult result = Utils.getTransactionResult(
                service, txHash, Constants.DEFAULT_WAITING_TIME);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        assertEquals(prevToBal.add(val), service.getBalance(to).execute());
    }

    public static void transferAndCheck(IconService service, Env.Chain chain,
                                        KeyWallet from, Address []to, BigInteger val) throws Exception{
        BigInteger prevFromBal = service.getBalance(from.getAddress()).execute();
        BigInteger expectedFromBal = prevFromBal.subtract(val.multiply(BigInteger.valueOf(to.length)));
        assertTrue(expectedFromBal.signum() >= 0);

        BigInteger []prevBal = new BigInteger[to.length];
        Bytes []txHash = new Bytes[to.length];
        for(int i = 0; i < to.length; i++) {
            prevBal[i] = service.getBalance(to[i]).execute();
            txHash[i] = Utils.transfer(service, chain.networkId,
                    from, to[i], val);
        }

        for(int i = 0; i < to.length; i++) {
            TransactionResult result = Utils.getTransactionResult(
                    service, txHash[i], Constants.DEFAULT_WAITING_TIME);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            assertEquals(prevBal[i].add(val), service.getBalance(to[i]).execute());
        }
    }
}
