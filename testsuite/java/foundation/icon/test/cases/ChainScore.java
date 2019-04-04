package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.HelloWorld;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.Parameterized;

import java.math.BigInteger;
import java.util.*;

@RunWith(Parameterized.class)
public class ChainScore {
    public static final Address CHAINSCORE_ADDRESS
            = new Address("cx0000000000000000000000000000000000000000");
    public static final Address GOV_ADDRESS
            = new Address("cx0000000000000000000000000000000000000001");
    public Address toAddr;
    private static Env.Chain chain;
    private static IconService iconService;

    public ChainScore(ChainScoreInput input){
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
                new ChainScoreInput(CHAINSCORE_ADDRESS),
                new ChainScoreInput(GOV_ADDRESS)
        );
    }

    private static KeyWallet scoreOwnerWallet;
    private static KeyWallet[]testWallets;
    private static final int testWalletNum = 3;
    private static HelloWorld helloWorld;

    @BeforeClass
    public static void init() throws Exception{
        Env.Node node = Env.nodes[0];
        chain = Env.nodes[0].chains[0];
        iconService = new IconService(new HttpProvider(node.endpointUrl));
        String value = "99999";
        testWallets = new KeyWallet[testWalletNum];
        Bytes[]txHash = new Bytes[testWalletNum + 1];
        for (int i = 0; i < testWalletNum + 1; i++) {
            KeyWallet wallet = KeyWallet.create();
            try {
                txHash[i] = Utils.transferIcx(iconService, Env.nodes[0].chains[0].godWallet
                        , wallet.getAddress(), value);
            } catch (Exception ex) {
                System.out.println("Failed to transfer");
            }
            if (i < testWalletNum) {
                testWallets[i] = wallet;
            } else {
                scoreOwnerWallet = wallet;
            }
        }

        helloWorld = HelloWorld.mustDeploy(iconService,
                scoreOwnerWallet, BigInteger.valueOf(0));
    }

    private void govCall(KeyWallet from, Address to, String function,
                         Map<String, String> param) throws Exception{
        Set<String> keySet = param.keySet();
        RpcObject.Builder builder = new RpcObject.Builder();
        for(String k : keySet) {
            builder.put(k, new RpcValue(param.get(k)));
        }
        Score.sendTransaction(iconService, BigInteger.valueOf(0),
                from, to, function, builder.build(), null);
    }

    @Test
    public void disableEnableScore() throws Exception{
        if(!this.toAddr.equals(CHAINSCORE_ADDRESS)) {
            return;
        }

        KeyWallet newKeyWallet = testWallets[0];
        String []testFuncs = {"disableScore", "enableScore"};
        KeyWallet[]fromWallets = {
                newKeyWallet,
                scoreOwnerWallet
        };

        if (!helloWorld.invokeHello(newKeyWallet)) {
            throw new Exception();
        }

        for (String func : testFuncs) {
            for (KeyWallet wallet : fromWallets) {
                try {
                    Map<String, String> paramMap = new HashMap<>();
                    paramMap.put("address", helloWorld.getAddress().toString());
                    govCall(wallet, CHAINSCORE_ADDRESS, func, paramMap);
                }
                catch (Exception ex) {
                    throw ex;
                }

                boolean invokeOk = helloWorld.invokeHello(newKeyWallet);
                if(invokeOk) {
                    if(wallet == scoreOwnerWallet && func.compareTo("disableScore") == 0) {
                        throw new Exception();
                    }
                }

                if((wallet == scoreOwnerWallet && func.compareTo("disableScore") == 0)
                    || wallet != scoreOwnerWallet && func.compareTo("enableScore") == 0){
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

        RpcItem item = Score.icxCall(iconService, BigInteger.valueOf(0), fromWallet, CHAINSCORE_ADDRESS,
                "getRevision", null);

        BigInteger revision = item.asInteger();
        Map<String, String> paramMap = new HashMap<>();
        revision = revision.add(BigInteger.valueOf(100));
        paramMap.put("code", revision.toString());
        try {
            govCall(fromWallet, this.toAddr, "setRevision", paramMap);
        }
        catch (Exception ex) {
            if(this.toAddr.equals(CHAINSCORE_ADDRESS)) {
                return;
            }
            throw ex;
        }

        item = Score.icxCall(iconService, BigInteger.valueOf(0), fromWallet, CHAINSCORE_ADDRESS,
                "getRevision", null);
        BigInteger newRevision = item.asInteger();
        System.out.println("getRevision : " + newRevision + ", revision : " + revision);
        if (!revision.equals(newRevision)) {
            if(!this.toAddr.equals(CHAINSCORE_ADDRESS)) {
                throw new Exception("Failed to set Revision");
            }
        }
    }

    public void acceptScore() {
    }

    public void rejectScore() {
    }

    // test block / unblock score
    @Test
    public void blockUnblockScore() throws Exception {
        KeyWallet fromWallet = scoreOwnerWallet;

        if (!helloWorld.invokeHello(testWallets[1])) {
            throw new Exception();
        }
        // true is block, false is unblock
        String []funcs = {"blockScore", "unblockScore"};

        for (String func : funcs) {
            RpcObject params = new RpcObject.Builder()
                    .put("address", new RpcValue(helloWorld.getAddress().toString()))
                    .build();
            try {
                Map<String, String> paramMap = new HashMap<>();
                paramMap.put("address", helloWorld.getAddress().toString());
                govCall(fromWallet, this.toAddr, func, paramMap);
            }
            catch (Exception ex) {
                throw ex;
            }

            boolean invokeOk = helloWorld.invokeHello(fromWallet);
            if(this.toAddr.equals(GOV_ADDRESS) && func.compareTo("blockScore") == 0) {
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

    public void setStepPrice() {
    }

    public void setStepCost() {}

    public void setMaxStepLimit() {}

    public void setValidators() {}

    public void grantValidator() {}

    public void revokeValidator() {}

    @Test
    public void addRemoveMember() throws Exception{
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        RpcItem item = Score.icxCall(iconService, BigInteger.valueOf(0), wallet, CHAINSCORE_ADDRESS,
                "getMembers", params);
        RpcObject rpcObject = item.asObject();
        RpcArray rpcArray = null;
        RpcItem memberItem = null;
        // TODO make function
        if((memberItem = rpcObject.getItem("memberList")) != null) {
            rpcArray = memberItem.asArray();
            for(int i = 0; i < rpcArray.size(); i++) {
                if(rpcArray.get(i).asAddress().equals(wallet)) {
                    throw new Exception();
                }
            }
        }
        String []funcs = {"addMember", "removeMember"};
        for (String func : funcs) {
            try {
                Map<String, String> paramMap = new HashMap<>();
                paramMap.put("address", wallet.getAddress().toString());
                govCall(wallet, this.toAddr, func, paramMap);
                item = Score.icxCall(iconService, BigInteger.valueOf(0), wallet, CHAINSCORE_ADDRESS,
                        "getMembers", params);

                rpcObject = item.asObject();
                boolean bFound = false;
                if((memberItem = rpcObject.getItem("memberList")) != null) {
                    rpcArray = memberItem.asArray();
                    for(int i = 0; i < rpcArray.size(); i++) {
                        if(rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                            bFound = true;
                            break;
                        }
                    }
                }

                if(this.toAddr.equals(CHAINSCORE_ADDRESS)) {
                   if(bFound == true) {
                       throw new Exception();
                   }
                } else {
                    if(func.compareTo("addMember") == 0) {
                       if(bFound == false) {
                           throw new Exception();
                       }
                    }
                }
            }
            catch (Exception ex) {
                throw ex;
            }
        }
    }

    @Test
    public void addRemoveDeployer() throws Exception {
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        RpcItem item = Score.icxCall(iconService, BigInteger.valueOf(0), wallet, CHAINSCORE_ADDRESS,
                "isDeployer", params);
        if (item.asBoolean()) {
            throw new Exception();
        }
        String []funcs = {"addDeployer", "removeDeployer"};
        for (String func : funcs) {
            try {
                Map<String, String> paramMap = new HashMap<>();
                paramMap.put("address", wallet.getAddress().toString());
                govCall(wallet, this.toAddr, func, paramMap);
                item = Score.icxCall(iconService, BigInteger.valueOf(0), wallet, CHAINSCORE_ADDRESS,
                        "isDeployer", params);
                if(item.asBoolean()) {
                   if(func.compareTo("addDeployer") != 0 || !this.toAddr.equals(GOV_ADDRESS)) {
                       throw new Exception();
                   }
                }
            }
            catch (Exception ex) {
                throw ex;
            }
        }
    }

    public void addLicense() {}

    public void removeLicense( ) {}

}
