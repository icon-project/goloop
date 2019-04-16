package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.experimental.categories.Category;

import static org.junit.Assert.*;

import java.io.IOException;
import java.math.BigInteger;
import java.util.*;

/*
test cases
1. Not enough balance.
2. Not enough stepLimit.
3. Invalid signature
4. TransferTest coin. check balances of both accounts with GetBalance api.
 - Check balances in every transaction.
 - Check
 set StepPrice 0 or not.
 -
5.
 */
public class TransferTest {
    private static KeyWallet godWallet;
    private static KeyWallet[]testWallets;
    private static IconService iconService;
    private static Env.Chain chain;
    private static final int testWalletNum = 10;
    private static GovScore govScore;
    private static KeyWallet govWallet;

    @BeforeClass
    public static void init() throws Exception {
        testWallets = new KeyWallet[testWalletNum];
        for(int i = 0; i < testWallets.length; i++){
            testWallets[i] = KeyWallet.create();
        }
        Env.Node node = Env.nodes[0];
        chain = Env.nodes[0].chains[0];
        godWallet = chain.godWallet;
        iconService = new IconService(new HttpProvider(node.endpointUrl));
        govWallet = KeyWallet.create();
        govScore = new GovScore(iconService, Env.nodes[0].chains[0].networkId, govWallet);
        initTransfer(godWallet, testWallets, 1000);
        initTransfer(godWallet, new KeyWallet[]{govWallet}, 999999);
    }

    // add step
    public static void initTransfer(KeyWallet from, KeyWallet []to, long value) throws Exception {
        long total = to.length * value;
        Bytes txHash = null;
        if (iconService.getBalance(from.getAddress()).execute()
                .compareTo(BigInteger.valueOf(total)) < 0) {
            throw new Exception();
        }
        for (KeyWallet w : to) {
            txHash = Utils.transfer(iconService, from, w.getAddress(), value);
        }

        try {
            Utils.getTransactionResult(iconService, txHash, 3000);
        }
        catch (ResultTimeoutException ex) {
            System.out.println("Failed to transfer");
            throw ex;
        }
    }

    /*
    sendTransaction with no balance account.
    and call getTransactionResult with txHash
    If account has no enough balance, the transaction will be dropped.
     */
    @Test
    public void notEnoughBalance() throws Exception{
        KeyWallet[]wallets = new KeyWallet[5];
        Random rand = new Random();
        for (int i = 0; i < wallets.length; i++) {
            wallets[i] = KeyWallet.create();
            if (iconService.getBalance(wallets[i].getAddress()).execute()
                    .compareTo(BigInteger.valueOf(0)) != 0) {
                throw new Exception();
            }

            // transfer from GOD to test wallets
            BigInteger bal = iconService.getBalance(testWallets[i].getAddress()).execute();
            Bytes txHash = Utils.transfer(iconService, wallets[i], testWallets[i].getAddress(), rand.nextInt(100) + 1);

            try {
                Utils.getTransactionResult(iconService, txHash, 5000);
                throw new Exception();
            }
            catch (ResultTimeoutException ex) {
                // success
            }

            BigInteger bal2;
            if((iconService.getBalance(testWallets[i].getAddress()).execute()).compareTo(bal) != 0) {
                throw new Exception();
            }
        }
    }

    @Test
    public void notEnoughStepLimit() throws Exception{
        KeyWallet testWallet = KeyWallet.create();
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue("invoke"))
                .build();
        BigInteger maxStepLimit = Utils.icxCall(iconService, BigInteger.valueOf(0), govWallet, Constants.CHAINSCORE_ADDRESS,
                "getMaxStepLimit", params).asInteger();
        BigInteger stepCost = Utils.icxCall(iconService, BigInteger.valueOf(0), testWallet,
                Constants.CHAINSCORE_ADDRESS,"getStepCosts", null).asObject().getItem("default").asInteger();

        // transfer from GOD to test wallets
        final long defStep  = 10;
        final long sp = 10;
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000));
        govScore.setStepCost("default", BigInteger.valueOf(defStep));
        govScore.setStepPrice(BigInteger.valueOf(sp));

        final long value = 1;
        RpcItem item = Utils.icxCall(iconService, BigInteger.valueOf(0), testWallet,
                Constants.CHAINSCORE_ADDRESS,"getStepCosts", null);
        BigInteger stepDefault = item.asObject().getItem("default").asInteger();

        item = Utils.icxCall(iconService, BigInteger.valueOf(0), testWallet,
                Constants.CHAINSCORE_ADDRESS,"getStepPrice", null);
        BigInteger need = stepDefault.multiply(item.asInteger()).add(BigInteger.valueOf(1));

        long []limits = {0, defStep};
        for(long sl : limits) {
            for(KeyWallet wallet : testWallets) {
                BigInteger bal = iconService.getBalance(wallet.getAddress()).execute();
                if(bal.compareTo(need) > 0) {
                    Transaction transaction = TransactionBuilder.newBuilder()
                            .nid(Env.nodes[0].chains[0].networkId)
                            .from(wallet.getAddress())
                            .to(wallet.getAddress())
                            .value(BigInteger.valueOf(value))
                            .stepLimit(BigInteger.valueOf(sl))
                            .timestamp(Utils.getMicroTime())
                            .nonce(BigInteger.valueOf(1))
                            .build();

                    SignedTransaction signedTransaction = new SignedTransaction(transaction, wallet);
                    Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
                    try {
                        Utils.getTransactionResult(iconService, txHash, 3000);
                        if(sl == 0) {
                            throw new Exception();
                        }
                    }
                    catch (ResultTimeoutException ex) {
                        if(sl != 0) {
                            throw ex;
                        }
                    }
                }
            }
        }
        govScore.setStepCost("default", stepCost);
        govScore.setStepPrice(BigInteger.ZERO);
        govScore.setMaxStepLimit("invoke", maxStepLimit);
    }

    @Test
    public void invalidSignature() throws Exception {
        KeyWallet testWallet = KeyWallet.create();
        for(KeyWallet wallet : testWallets) {
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(Env.nodes[0].chains[0].networkId)
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
    }

    class Account {
        private KeyWallet wallet;
        private Bytes txHash;
        private BigInteger balance;
        private List<String> acRecord;

        Account(KeyWallet wallet, BigInteger initBal) {
            this.wallet = wallet;
            this.balance = initBal;
            acRecord = new LinkedList();
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
            txHash = Utils.transfer(iconService, wallet, account.getAddress(), value.longValue());
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
                assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
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
        int transferNum = 1000;
        final int testWalletNum = 1000;
        Account []testAccounts = new Account[testWalletNum];
        Account godAccount = new Account(godWallet,
                iconService.getBalance(godWallet.getAddress()).execute());
        Random rand = new Random();
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

            if(i == testWalletNum -1) {
                godAccount.checkBalance();
            }
        }

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

        for(Account account : testAccounts) {
            assertTrue(account.checkBalance());
        }
    }


    public void transferWithMessage() throws Exception {

    }
}
