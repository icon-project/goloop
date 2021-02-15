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

package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.Transaction;
import foundation.icon.icx.TransactionBuilder;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.ConfirmedTransaction;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.GovScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.nio.charset.StandardCharsets;
import java.util.LinkedList;
import java.util.List;
import java.util.Random;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.fail;

@Tag(Constants.TAG_PY_GOV)
@Tag(Constants.TAG_JAVA_GOV)
public class TransferTest extends TestBase {
    private static TransactionHandler txHandler;
    private static Env.Chain chain;
    private static KeyWallet[] testWallets;
    private static final int testWalletNum = 5;
    private static GovScore govScore;
    private static GovScore.Fee fee;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        govScore = new GovScore(txHandler);
        fee = govScore.getFee();

        testWallets = new KeyWallet[testWalletNum];
        Address[] addrs = new Address[testWalletNum];
        for (int i = 0; i < testWalletNum; i++) {
            KeyWallet wallet = KeyWallet.create();
            testWallets[i] = wallet;
            addrs[i] = wallet.getAddress();
        }
        transferAndCheckResult(txHandler, addrs, ICX);
        transferAndCheckResult(txHandler, chain.governorWallet.getAddress(), ICX);
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    /*
    sendTransaction with no balance account.
    and call getTransactionResult with txHash
    If account has no enough balance, the transaction will not be executed.
     */
    @Test
    public void notEnoughBalance() throws Exception {
        LOG.infoEntering("notEnoughBalance");
        KeyWallet[] wallets = new KeyWallet[testWalletNum];
        Bytes[] hashes = new Bytes[wallets.length];
        BigInteger[] balances = new BigInteger[wallets.length];
        Random rand = new Random();
        for (int i = 0; i < wallets.length; i++) {
            wallets[i] = KeyWallet.create();
            balances[i] = txHandler.getBalance(testWallets[i].getAddress());
            // transfer from no balance wallet to test wallets
            hashes[i] = txHandler.transfer(wallets[i], testWallets[i].getAddress(),
                    BigInteger.valueOf(rand.nextInt(100) + 1), Constants.DEFAULT_STEPS);
        }
        for (int i = 0; i < hashes.length; i++) {
            try {
                long waitingTime = 5000;
                if (i != 0) {
                    waitingTime = 500;
                }
                txHandler.getResult(hashes[i], waitingTime);
                fail();
            } catch (ResultTimeoutException e) {
                // success
                LOG.info("Expected exception: msg=" + e.getMessage());
            }
            assertEquals(balances[i], txHandler.getBalance(testWallets[i].getAddress()));
        }
        LOG.infoExiting();
    }

    @Test
    public void notEnoughStepLimit() throws Exception {
        LOG.infoEntering("notEnoughStepLimit");
        ChainScore chainScore = new ChainScore(txHandler);
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue("invoke"))
                .build();
        BigInteger prevMaxStepLimit = chainScore.call("getMaxStepLimit", params).asInteger();
        BigInteger prevDefStepCost = chainScore.call("getStepCosts", null).asObject().getItem("default").asInteger();
        BigInteger prevStepPrice = chainScore.getStepPrice();

        final long newDefStepCost = 100;
        final long newStepPrice = 10;
        LOG.infoEntering("setup", "new stepCosts");
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(10000));
        govScore.setStepCost("default", BigInteger.valueOf(newDefStepCost));
        govScore.setStepPrice(BigInteger.valueOf(newStepPrice));
        LOG.infoExiting();

        KeyWallet testWallet = testWallets[0];
        KeyWallet toWallet = KeyWallet.create();
        long[] limits = {0, 1, newDefStepCost - 1, newDefStepCost};
        Bytes[] hashes = new Bytes[limits.length];
        final BigInteger testValue = BigInteger.ONE;
        int cnt = 0;
        for (long testLimit : limits) {
            LOG.infoEntering("invoke", "required[" + newDefStepCost + "], set[" + testLimit + "]");
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(testWallet.getAddress())
                    .to(toWallet.getAddress())
                    .value(testValue)
                    .stepLimit(BigInteger.valueOf(testLimit))
                    .build();
            hashes[cnt++] = txHandler.invoke(testWallet, transaction);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            try {
                txHandler.getResult(hashes[i]);
                assertEquals(newDefStepCost, limits[i]);
                BigInteger resultBal = txHandler.getBalance(toWallet.getAddress());
                assertEquals(testValue, resultBal);
            } catch (RpcError e) {
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                assertNotEquals(newDefStepCost, limits[i]);
            } catch (ResultTimeoutException e) {
                LOG.info("Expected Timeout: msg=" + e.getMessage());
                assertNotEquals(newDefStepCost, limits[i]);
            } catch (Exception e) {
                fail(e);
            }
        }

        LOG.infoEntering("restore", "stepCosts");
        govScore.setStepPrice(prevStepPrice);
        govScore.setStepCost("default", prevDefStepCost);
        govScore.setMaxStepLimit("invoke", prevMaxStepLimit);
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void invalidSignature() throws Exception {
        LOG.infoEntering("invalidSignature");
        KeyWallet testWallet = KeyWallet.create();
        for (KeyWallet wallet : testWallets) {
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(wallet.getAddress())
                    .to(testWallet.getAddress())
                    .value(BigInteger.ONE)
                    .stepLimit(BigInteger.ONE)
                    .build();
            try {
                txHandler.invoke(KeyWallet.create(), transaction);
                fail();
            } catch (RpcError e) {
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
            }
        }
        LOG.infoExiting();
    }

    static class Account {
        private KeyWallet wallet;
        private Bytes txHash;
        private BigInteger balance;
        private List<String> acRecord;

        Account(KeyWallet wallet, BigInteger initBal) {
            this.wallet = wallet;
            this.balance = initBal;
            acRecord = new LinkedList<>();
            acRecord.add("initBal : " + initBal);
        }

        Address getAddress() {
            return wallet.getAddress();
        }

        void receive(BigInteger value, Bytes txHash) {
            balance = balance.add(value);
            this.txHash = txHash;
            acRecord.add("received " + value + ", current balance : " + balance);
        }

        // return false if not enough balance
        boolean transfer(Account account, BigInteger value) throws Exception {
            if (balance.compareTo(value) < 0) {
                return false;
            }
            txHash = txHandler.transfer(wallet, account.getAddress(), value, Constants.DEFAULT_STEPS);
            balance = balance.subtract(value);
            acRecord.add("transfer " + value + ", current balance : " + balance);
            account.receive(value, txHash);
            return true;
        }

        void printRecord() {
            for (String record : acRecord) {
                System.out.println(record);
            }
        }

        boolean checkBalance() {
            try {
                if (txHash == null) {
                    return true;
                }
                assertSuccess(txHandler.getResult(txHash));
                BigInteger cmpBal = txHandler.getBalance(wallet.getAddress());
                if (cmpBal.compareTo(balance) != 0) {
                    System.out.println("calculated balance " + balance + ", getBalance " + cmpBal);
                    printRecord();
                    return false;
                }
            } catch (Exception e) {
                fail(e);
            }
            return true;
        }

        BigInteger getBalance() {
            return balance;
        }
    }

    @Test
    public void transferAndCheckBal() throws Exception {
        LOG.infoEntering("transferAndCheckBal");
        int transferNum = 1000;
        final int testWalletNum = 1000;
        Account[] testAccounts = new Account[testWalletNum];
        Account godAccount = new Account(chain.godWallet, txHandler.getBalance(chain.godWallet.getAddress()));
        Random rand = new Random();
        LOG.infoEntering("transfer", "from god to test addresses");
        for (int i = 0; i < testWalletNum; i++) {
            KeyWallet wallet = KeyWallet.create();
            BigInteger value;
            do {
                value = BigInteger.valueOf(rand.nextInt(Integer.MAX_VALUE));
                testAccounts[i] = new Account(wallet, BigInteger.ZERO);
            } while (!godAccount.transfer(testAccounts[i], value));
        }
        assertTrue(godAccount.checkBalance());
        LOG.infoExiting();

        LOG.infoEntering("transfer", "from test address to another");
        while (transferNum > 0) {
            int from , to, value;
            do {
                from = rand.nextInt(testWalletNum);
            } while (testAccounts[from].getBalance().compareTo(BigInteger.ZERO) == 0);

            do {
                to = rand.nextInt(testWalletNum);
            } while (from == to);

            BigInteger bal = testAccounts[from].getBalance();
            value = rand.nextInt(bal.compareTo(
                    BigInteger.valueOf(Integer.MAX_VALUE)) > 0
                    ? BigInteger.valueOf(Integer.MAX_VALUE).intValue()
                    : bal.intValue());
            testAccounts[from].transfer(testAccounts[to], BigInteger.valueOf(value));
            transferNum--;
        }
        LOG.infoExiting();

        LOG.infoEntering("check", "balances");
        for (Account account : testAccounts) {
            assertTrue(account.checkBalance());
        }
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void transferWithMessage() throws Exception {
        LOG.infoEntering("transferWithMessage");
        KeyWallet testWallet = KeyWallet.create();
        String[] msgs = new String[testWallets.length];
        Bytes[] hashes = new Bytes[testWallets.length];
        int cnt = 0;
        for (KeyWallet wallet : testWallets) {
            LOG.infoEntering("invoke", "from " + wallet.getAddress());
            String msg = "message: " + wallet.toString();
            int stepLimit = (msg.length() + 2) * 2 + 1;
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(wallet.getAddress())
                    .to(testWallet.getAddress())
                    .value(BigInteger.ONE)
                    .stepLimit(BigInteger.valueOf(stepLimit))
                    .message(msg)
                    .build();
            msgs[cnt] = msg;
            hashes[cnt++] = txHandler.invoke(wallet, transaction);
            LOG.infoExiting();
        }
        for (int i = 0; i < cnt; i++) {
            LOG.infoEntering("check", "msg i=" + i);
            txHandler.getResult(hashes[i]);
            ConfirmedTransaction tx = txHandler.getTransaction(hashes[i]);
            StringBuilder sb = new StringBuilder("0x");
            for (byte b : msgs[i].getBytes(StandardCharsets.UTF_8)) {
                sb.append(String.format("%02x", b));
            }
            assertEquals(sb.toString(), tx.getData().asBytes().toString());
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }
}
