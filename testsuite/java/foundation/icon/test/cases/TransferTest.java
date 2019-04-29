package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import org.junit.BeforeClass;
import org.junit.Test;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.Assert.*;

import java.io.IOException;
import java.math.BigInteger;
import java.util.*;
import java.util.stream.Collectors;

/*
test cases
1. Not enough balance.
2. Not enough stepLimit.
3. Invalid signature
coin. check balances of both accounts with GetBalance api.
 - Check balances in every transaction.
 - Check
 set StepPrice 0 or not.
 -
5.
 */
public class TransferTest {
    private static KeyWallet[]testWallets;
    private static IconService iconService;
    private static Env.Chain chain;
    private static final int testWalletNum = 10;
    private static GovScore govScore;

    @BeforeClass
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        govScore = new GovScore(iconService, chain);
        initTransfer();
    }

    // add step
    public static void initTransfer() throws Exception {
        testWallets = new KeyWallet[testWalletNum];
        Address []addrs = new Address[testWalletNum];
        for(int i = 0; i < testWalletNum; i++){
            KeyWallet wallet = KeyWallet.create();
            testWallets[i] = wallet;
            addrs[i] = wallet.getAddress();
        }

        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);
        Utils.transferAndCheck(iconService, chain, chain.godWallet, chain.governorWallet.getAddress(), Constants.DEFAULT_BALANCE);
    }

    /*
    sendTransaction with no balance account.
    and call getTransactionResult with txHash
    If account has no enough balance, the transaction will not be executed.
     */
    @Test
    public void notEnoughBalance() throws Exception{
        LOG.infoEntering( "notEnoughBalance");
        KeyWallet[]wallets = new KeyWallet[5];
        Random rand = new Random();
        for (int i = 0; i < wallets.length; i++) {
            wallets[i] = KeyWallet.create();
            Utils.assertEquals(BigInteger.ZERO, iconService.getBalance(wallets[i].getAddress()).execute());

            // transfer from no balance wallet to test wallets
            BigInteger bal = iconService.getBalance(testWallets[i].getAddress()).execute();
            Bytes txHash = Utils.transfer(iconService, chain.networkId, wallets[i], testWallets[i].getAddress(), rand.nextInt(100) + 1);
            try {
                Utils.getTransactionResult(iconService, txHash, 5000);
                fail();
            }
            catch (ResultTimeoutException ex) {
                // success
            }
            Utils.assertEquals(bal, iconService.getBalance(testWallets[i].getAddress()).execute());
        }
        LOG.infoExiting();
    }

    @Test
    public void notEnoughStepLimit() throws Exception{
        LOG.infoEntering( "notEnoughStepLimit");
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue("invoke"))
                .build();
        BigInteger prevMaxStepLimit = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getMaxStepLimit", params).asInteger();
        BigInteger prevDefStepCost = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS,"getStepCosts", null)
                .asObject().getItem("default").asInteger();
        BigInteger prevStepPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS,"getStepPrice", null)
                .asInteger();

        final long newDefStepCost  = 10;
        final long newStepPrice = 10;
        LOG.infoEntering( "setMaxStepLimit");
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000));
        LOG.infoExiting();
        LOG.infoEntering( "setStepCost");
        govScore.setStepCost("default", BigInteger.valueOf(newDefStepCost));
        LOG.infoExiting();
        LOG.infoEntering( "setStepPrice");
        govScore.setStepPrice(BigInteger.valueOf(newStepPrice));
        LOG.infoExiting();

        KeyWallet testWallet = testWallets[0];
        KeyWallet toWallet = KeyWallet.create();
        long []limits = {0, 1, newDefStepCost - 1, newDefStepCost};
        final BigInteger testValue = BigInteger.ONE;
        for(long testLimit : limits) {
            BigInteger toBal = iconService.getBalance(toWallet.getAddress()).execute();
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(BigInteger.valueOf(chain.networkId))
                    .from(testWallet.getAddress())
                    .to(toWallet.getAddress())
                    .value(testValue)
                    .stepLimit(BigInteger.valueOf(testLimit))
                    .timestamp(Utils.getMicroTime())
                    .nonce(BigInteger.ONE)
                    .build();

            SignedTransaction signedTransaction = new SignedTransaction(transaction, testWallet);
            LOG.infoEntering("sendTransaction required[" + newDefStepCost + "], set[" + testLimit + "]");
            Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
            try {
                Utils.getTransactionResult(iconService, txHash, 5000);
                LOG.infoExiting();
                Utils.assertEquals(newDefStepCost, testLimit);
                BigInteger resultBal = iconService.getBalance(toWallet.getAddress()).execute();
                Utils.assertEquals(toBal.add(testValue), resultBal);
            }
            catch (ResultTimeoutException ex) {
                LOG.info("FAIL to get result");
                LOG.infoExiting();
                Utils.assertNotEquals(newDefStepCost, testLimit);
            }
        }
        govScore.setStepCost("default", prevDefStepCost);
        govScore.setStepPrice(prevStepPrice);
        govScore.setMaxStepLimit("invoke", prevMaxStepLimit);
        LOG.infoExiting();
    }

    @Test
    public void invalidSignature() throws Exception {
        LOG.infoEntering( "invalidSignature");
        KeyWallet testWallet = KeyWallet.create();
        for(KeyWallet wallet : testWallets) {
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(BigInteger.valueOf(chain.networkId))
                    .from(wallet.getAddress())
                    .to(testWallet.getAddress())
                    .value(BigInteger.valueOf(1))
                    .stepLimit(BigInteger.valueOf(1))
                    .timestamp(Utils.getMicroTime())
                    .nonce(BigInteger.valueOf(1))
                    .build();

            SignedTransaction signedTransaction = new SignedTransaction(transaction, KeyWallet.create());
            try {
                iconService.sendTransaction(signedTransaction).execute();
            }
            catch(IOException ex) {
                continue;
            }
            fail();
        }
        LOG.infoExiting();
    }

    class Account {
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
            if(balance.compareTo(value) < 0) {
                return false;
            }
            txHash = Utils.transfer(iconService, chain.networkId, wallet, account.getAddress(), value.longValue());
            balance = balance.subtract(value);
            acRecord.add("transfer " + value + ", current balance : " + balance);
            account.receive(value, txHash);
            return true;
        }

        void printRecord() {
            for(String record : acRecord) {
                System.out.println(record);
            }
        }

        boolean checkBalance() {
            try {
                if(txHash == null) {
                    return true;
                }
                TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
                Utils.assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
                BigInteger cmpBal = iconService.getBalance(wallet.getAddress()).execute();
                if(cmpBal.compareTo(balance) != 0){
                    System.out.println("calculated balance " + balance + ", getBalance " + cmpBal);
                    printRecord();
                    return false;
                }
            }
            catch (Exception ex) {
                fail();
            }
            return true;
        }
        BigInteger getBalance() {
            return balance;
        }
    }

    @Test
    public void transferAndCheckBal() throws Exception {
        LOG.infoEntering( "transferAndCheckBal");
        int transferNum = 1000;
        final int testWalletNum = 1000;
        Account []testAccounts = new Account[testWalletNum];
        Account godAccount = new Account(chain.godWallet,
                iconService.getBalance(chain.godWallet.getAddress()).execute());
        Random rand = new Random();
        LOG.infoEntering( "transfer from god to test addresses");
        for(int i = 0; i < testWalletNum; i++) {
            KeyWallet wallet;
            BigInteger value;
            do {
                 wallet = KeyWallet.create();
            } while(iconService.getBalance(wallet.getAddress()).execute().compareTo(BigInteger.ZERO) != 0);

            do {
                value = BigInteger.valueOf(rand.nextInt(Integer.MAX_VALUE));
                testAccounts[i] = new Account(wallet, BigInteger.ZERO);
            } while(!godAccount.transfer(testAccounts[i], value));
        }
        assertTrue(godAccount.checkBalance());
        LOG.infoExiting();

        LOG.infoEntering( "transfer from test address to another");
        while(transferNum > 0) {
            int from , to, value;
            do {
                from = rand.nextInt(testWalletNum);
            } while(testAccounts[from].getBalance().compareTo(BigInteger.ZERO) == 0);

            do {
                to = rand.nextInt(testWalletNum);
            }while(from == to);

            BigInteger bal = testAccounts[from].getBalance();

            value = rand.nextInt(bal.compareTo(BigInteger.valueOf(Integer.MAX_VALUE)) > 0 ? BigInteger.valueOf(Integer.MAX_VALUE).intValue() : bal.intValue());
            testAccounts[from].transfer(testAccounts[to], BigInteger.valueOf(value));
            transferNum--;
        }
        LOG.infoExiting();

        for(Account account : testAccounts) {
            assertTrue(account.checkBalance());
        }
        LOG.infoExiting();
    }


    public void transferWithMessage() throws Exception {

    }
}
