package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.HelloWorld;
import org.junit.BeforeClass;
import org.junit.Ignore;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.Parameterized;
import java.math.BigInteger;
import java.util.*;

@RunWith(Parameterized.class)
public class ChainScoreTest {
    public Address toAddr;
    private static Env.Chain chain;
    private static IconService iconService;

    public ChainScoreTest(ChainScoreInput input){
        toAddr = input.getTo();
    }

    static class ChainScoreInput {
        private Address to;
        public ChainScoreInput(Address to) {
            this.to = to;
        }
        public Address getTo() {
            return this.to;
        }
    }

    @Parameterized.Parameters
    public static Collection<ChainScoreInput> initInput() {
        return Arrays.asList(
                new ChainScoreInput(Constants.CHAINSCORE_ADDRESS),
                new ChainScoreInput(Constants.GOV_ADDRESS)
        );
    }

    private static KeyWallet helloWorldOwner;
    private static KeyWallet[]testWallets;
    private static final int testWalletNum = 3;
    private static HelloWorld helloWorld;
    private static boolean isAuditEnabled;

    @BeforeClass
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        chain = Env.nodes[0].chains[0];
        iconService = new IconService(new HttpProvider(node.endpointUrl));
        testWallets = new KeyWallet[testWalletNum];
        initChainScore();
    }

    static void initChainScore() throws Exception {
        long value = 999999999;
        Bytes[]txHash = new Bytes[testWalletNum + 1];
        for (int i = 0; i < testWalletNum + 1; i++) {
            KeyWallet wallet = KeyWallet.create();
            try {
                txHash[i] = Utils.transfer(iconService, Env.nodes[0].chains[0].godWallet
                        , wallet.getAddress(), value);
            } catch (Exception ex) {
                System.out.println("Failed to transfer");
            }
            if (i < testWalletNum) {
                testWallets[i] = wallet;
            } else {
                helloWorldOwner = wallet;
            }
        }

        helloWorld = HelloWorld.mustDeploy(iconService,
                helloWorldOwner, BigInteger.valueOf(0));
    }

    @Test
    public void disableEnableScore() throws Exception{
        if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
            return;
        }

        KeyWallet newKeyWallet = testWallets[0];
        String []methods = {"disableScore", "enableScore"};
        KeyWallet[]fromWallets = {
                newKeyWallet,
                helloWorldOwner
        };

        if (!helloWorld.invokeHello(newKeyWallet)) {
            throw new Exception();
        }

        for (String method : methods) {
            for (KeyWallet wallet : fromWallets) {
                try {
                    RpcObject.Builder builder = new RpcObject.Builder();
                    builder.put("address", new RpcValue(helloWorld.getAddress().toString()));
                    TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                            wallet, Constants.CHAINSCORE_ADDRESS, method, builder.build(), 0);
                    if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                        if(wallet.equals(helloWorldOwner)) {
                            throw new Exception();
                        }
                    }
                }
                catch (Exception ex) {
                    if(wallet.equals(helloWorldOwner)) {
                        throw ex;
                    }
                }

                boolean invokeOk = helloWorld.invokeHello(newKeyWallet);
                if((wallet == helloWorldOwner && method.compareTo("disableScore") == 0)
                    || (wallet != helloWorldOwner && method.compareTo("enableScore") == 0)){
                    if(invokeOk) {
                        throw new Exception();
                    }
                } else {
                    if(!invokeOk) {
                        throw new Exception();
                    }
                }
            }
        }
    }

    @Test
    public void setRevision() throws Exception{
        KeyWallet fromWallet = KeyWallet.create();

        RpcItem item = Utils.icxCall(iconService, BigInteger.valueOf(0), fromWallet, Constants.CHAINSCORE_ADDRESS,
                "getRevision", null);

        BigInteger revision = item.asInteger();
        try {
            revision = revision.add(BigInteger.valueOf(100));
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("code", new RpcValue(revision.toString()));
            TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                    fromWallet, this.toAddr, "setRevision", builder.build(), 0);
            if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                    throw new Exception();
                }
            }
        }
        catch (Exception ex) {
            if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                return;
            }
            throw ex;
        }

        BigInteger newRevision = Utils.icxCall(iconService, BigInteger.valueOf(0), fromWallet, Constants.CHAINSCORE_ADDRESS,
                "getRevision", null).asInteger();
        if (!revision.equals(newRevision)) {
            if(!this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                throw new Exception("Failed to set Revision");
            }
        }
    }

    @Test
    public void acceptScore() throws Exception{
        if (!isAuditEnabled) {
            return;
        }
        Bytes txHash = Utils.deployScore(iconService, helloWorldOwner, Constants.SCORE_ROOT + "helloWorld.zip", null, 10000);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new Exception();
        }
        Address addr = new Address(result.getScoreAddress());
        KeyWallet tmpWallet = KeyWallet.create();
        boolean []list = {false, true};
        for(boolean accepted : list) {
            try {
                Utils.sendTransactionWithCall(iconService, BigInteger.ONE,
                        tmpWallet, addr, "hello", null, 0);
                if(!accepted) {
                    throw new Exception();
                }
            }
            catch(ResultTimeoutException ex) {
                if(accepted) {
                    throw new Exception();
                }
            }
            if(!accepted) {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("txHash", new RpcValue(txHash.toHexString(true)));
                Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        tmpWallet, this.toAddr, "acceptScore", builder.build(), 0);
            }
        }
    }

    @Test
    public void rejectScore() throws Exception {
        if (!isAuditEnabled) {
            return;
        }
        Bytes txHash = Utils.deployScore(iconService, helloWorldOwner, Constants.SCORE_ROOT + "helloWorld.zip", null, 10000);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new Exception();
        }
        Address addr = new Address(result.getScoreAddress());
        KeyWallet tmpWallet = KeyWallet.create();
        boolean []list = {false, true};
        for(boolean rejected : list) {
            try {
                Utils.sendTransactionWithCall(iconService, BigInteger.ONE,
                        tmpWallet, addr, "hello", null, 0);
                throw new Exception();
            }
            catch(ResultTimeoutException ex) {
            }
            if(!rejected) {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("txHash", new RpcValue(txHash.toHexString(true)));
                Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        tmpWallet, this.toAddr, "rejectScore", builder.build(), 0);
            } else {
                // TODO get scoreStatus
            }
        }
    }

    // test block / unblock score
    @Test
    public void blockUnblockScore() throws Exception {
        KeyWallet fromWallet = helloWorldOwner;

        try {
            if (!helloWorld.invokeHello(testWallets[1])) {
                throw new Exception();
            }
        }
        catch (ResultTimeoutException ex) {
            throw new Exception();
        }
        // true is block, false is unblock
        String []methods = {"blockScore", "unblockScore"};

        for (String method : methods) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("address", new RpcValue(helloWorld.getAddress().toString()));
                TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        fromWallet, this.toAddr, method, builder.build(), 0);
                if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                    if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                }
            }
            catch (Exception ex) {
                if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                    return;
                }
                else {
                    throw ex;
                }
            }

            boolean invokeOk = helloWorld.invokeHello(fromWallet);
            if(this.toAddr.equals(Constants.GOV_ADDRESS) && method.compareTo("blockScore") == 0) {
                if(invokeOk) {
                    throw new Exception();
                }
            } else {
               if(!invokeOk) {
                   throw new Exception();
               }
            }
        }
    }

    @Test
    public void setStepPrice() throws Exception {
        KeyWallet wallet = testWallets[0];
        BigInteger stepPrice = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                "getStepPrice", null).asInteger();

        stepPrice = stepPrice.add(BigInteger.valueOf(10));
        BigInteger []stepPrices = new BigInteger[] {
                stepPrice, BigInteger.ZERO
        };
        for(BigInteger price : stepPrices) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("stepPrice", new RpcValue(price.toString()));
                TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        wallet, this.toAddr, "setStepPrice", builder.build(), 0);
                if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                    if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                }
                if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                    throw new Exception();
                }
                BigInteger resultPrice = Utils.icxCall(iconService, BigInteger.valueOf(0),
                        wallet, Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
                if(resultPrice.compareTo(price) != 0) {
                    throw new Exception();
                }
            }
            catch(Exception ex) {
                if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                    throw ex;
                }
            }
        }
    }

    @Test
    public void setStepCost() throws Exception{
        KeyWallet wallet = testWallets[0];
        RpcItem stepCosts = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                "getStepCosts", null);
        Bytes []txHashList = new Bytes[GovScore.stepCostTypes.length];
        String []cTypes = {"invoke", "query"};
        for(String cType : cTypes) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("contextType", new RpcValue(cType));
                builder.put("value", new RpcValue("10000000"));
                TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        wallet, this.toAddr, "setMaxStepLimit", builder.build(), 0);
                if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                    if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                }
            }
            catch(Exception ex) {
                if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                    continue;
                }
            }
        }

        Map<String, BigInteger> originMap = new HashMap<>();
        Map<String, BigInteger> costMap = new HashMap<>();
        RpcObject rpcObject = stepCosts.asObject();
        long cnt = 1;
        for(int i = 0; i < GovScore.stepCostTypes.length; i++) {
            String type = GovScore.stepCostTypes[i];
            BigInteger oCost = rpcObject.getItem(type).asInteger();
            originMap.put(type, oCost);

            BigInteger bCost = oCost.add(BigInteger.valueOf(cnt));
            costMap.put(type, bCost);
            cnt += 1;
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("stepType", new RpcValue(type));
                builder.put("cost", new RpcValue(bCost.toString()));
                txHashList[i] = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        wallet, this.toAddr, "setStepCost", builder.build(), 0, false);
            }
            catch(Exception ex) {
                if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                    throw new Exception();
                }
                else {
                    continue;
                }
            }
        }

        for(Bytes txHash : txHashList) {
            try {
                TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
                if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                    if (this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                }
            }
            catch(ResultTimeoutException ex) {
                if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                    throw ex;
                }
                else {
                    break;
                }
            }
        }

        if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
            stepCosts = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                    "getStepCosts", null);
            rpcObject = stepCosts.asObject();
            for(String type : GovScore.stepCostTypes) {
                if(costMap.get(type).compareTo(rpcObject.getItem(type).asInteger()) != 0) {
                    throw new Exception();
                }
            }

            // rollback
            txHashList = new Bytes[GovScore.stepCostTypes.length];
            for(int i = 0; i < GovScore.stepCostTypes.length; i++) {
                String type = GovScore.stepCostTypes[i];
                try {
                    RpcObject.Builder builder = new RpcObject.Builder();
                    builder.put("stepType", new RpcValue(type));
                    builder.put("cost", new RpcValue(originMap.get(type)));
                    txHashList[i] = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                            wallet, this.toAddr, "setStepCost", builder.build(), 0, false);
                }
                catch(Exception ex) {
                    if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                    else {
                        continue;
                    }
                }
            }

            for(Bytes txHash : txHashList) {
                try {
                    TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
                    if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                        throw new Exception();
                    }
                }
                catch(ResultTimeoutException ex) {
                    throw ex;
                }
            }
        }

        for(String type : GovScore.stepCostTypes) {
            cnt += 1;
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("stepType", new RpcValue(type));
                builder.put("cost", new RpcValue(originMap.get(type).toString()));
                TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        wallet, this.toAddr, "setStepCost", builder.build(), 0);
                if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                    if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                }
            }
            catch(Exception ex) {
                if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                    throw new Exception();
                }
                else {
                    continue;
                }
            }
        }
    }

    @Test
    public void setMaxStepLimit() throws Exception {
        KeyWallet wallet = KeyWallet.create();
        String []contextTypes = {"invoke", "query"};
        for(String type : contextTypes) {
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .build();
            BigInteger stepPrice = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                    "getMaxStepLimit", params).asInteger();

            stepPrice = stepPrice.add(BigInteger.valueOf(100));
            BigInteger []stepPrices = new BigInteger[] {
                    stepPrice, BigInteger.ZERO
            };
            for(BigInteger price : stepPrices) {
                try {
                    RpcObject.Builder builder = new RpcObject.Builder();
                    builder.put("contextType", new RpcValue(type));
                    builder.put("value", new RpcValue(price.toString()));
                    TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                            wallet, this.toAddr, "setMaxStepLimit", builder.build(), 0);
                    if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                        if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                            throw new Exception();
                        }
                    }

                    BigInteger resultPrice = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                            "getMaxStepLimit", params).asInteger();
                    System.out.println("price : " + price.toString() + ", result : " + resultPrice.toString() + ": toAddr = " + this.toAddr.toString());
                    if(resultPrice.compareTo(price) != 0) {
                        throw new Exception();
                    }
                }
                catch (Exception ex) {
                    if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                        continue;
                    }
                    else {
                        throw ex;
                    }
                }
            }
        }
    }

    // TBD : setValidator API
//    public void setValidators() {}
    @Ignore
    @Test
    public void grantRevokeValidator() throws Exception {
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        RpcItem item = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                "getValidators", params);
        RpcArray rpcArray = item.asArray();
        for(int i = 0; i < rpcArray.size(); i++) {
            if(rpcArray.get(i).asAddress().equals(wallet)) {
                throw new Exception();
            }
        }
        String []funcs = {"grantValidator", "revokeValidator"};
        for (String func : funcs) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("address", new RpcValue(wallet.getAddress().toString()));
                TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        wallet, this.toAddr, func, builder.build(), 0);
                if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                    if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                }

                item = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                        "getValidators", params);

                boolean bFound = false;
                rpcArray = item.asArray();
                for(int i = 0; i < rpcArray.size(); i++) {
                    if(rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                        bFound = true;
                        break;
                    }
                }

                if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                    if(bFound == true) {
                        throw new Exception();
                    }
                } else {
                    if(func.compareTo("grantValidator") == 0) {
                        if(bFound == false) {
                            throw new Exception();
                        }
                    }
                }
            }
            catch (Exception ex) {
                System.out.println("toAddr : " + this.toAddr);
                if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                    continue;
                }
                else {
                    throw ex;
                }
            }
        }
    }

    @Test
    public void addRemoveMember() throws Exception{
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        RpcItem item = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                "getMembers", params);
        RpcArray rpcArray = item.asArray();
        for(int i = 0; i < rpcArray.size(); i++) {
            if(rpcArray.get(i).asAddress().equals(wallet)) {
                throw new Exception();
            }
        }
        String []methods = {"addMember", "removeMember"};
        for (String method : methods) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("address", new RpcValue(wallet.getAddress().toString()));
                TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        wallet, this.toAddr, method, builder.build(), 0);
                if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                    if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                }
                item = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                        "getMembers", params);

                boolean bFound = false;
                rpcArray = item.asArray();
                for(int i = 0; i < rpcArray.size(); i++) {
                    if(rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                        bFound = true;
                        break;
                    }
                }

                if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                   if(bFound == true) {
                       throw new Exception();
                   }
                } else {
                    if(method.compareTo("addMember") == 0) {
                       if(bFound == false) {
                           throw new Exception();
                       }
                    }
                }
            }
            catch (Exception ex) {
                if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                    return;
                }
                else {
                    throw ex;
                }
            }
        }
    }

    @Test
    public void addRemoveDeployer() throws Exception {
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        RpcItem item = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                "isDeployer", params);
        if (item.asBoolean()) {
            throw new Exception();
        }
        String []methods = {"addDeployer", "removeDeployer"};
        for (String method : methods) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("address", new RpcValue(wallet.getAddress().toString()));
                TransactionResult result = Utils.sendTransactionWithCall(iconService, BigInteger.valueOf(0),
                        wallet, this.toAddr, method, builder.build(), 0);
                if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                    if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                        throw new Exception();
                    }
                }

                item = Utils.icxCall(iconService, BigInteger.valueOf(0), wallet, Constants.CHAINSCORE_ADDRESS,
                        "isDeployer", params);
                if(item.asBoolean()) {
                   if(method.compareTo("addDeployer") != 0 || !this.toAddr.equals(Constants.GOV_ADDRESS)) {
                       throw new Exception();
                   }
                }
            }
            catch (Exception ex) {
                if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                    return;
                }
                else {
                    throw ex;
                }
            }
        }
    }

    public void addLicense() {}

    public void removeLicense( ) {}

}
