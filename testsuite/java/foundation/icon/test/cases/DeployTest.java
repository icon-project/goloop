package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.*;
import foundation.icon.test.score.GovScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.IOException;
import java.math.BigInteger;
import java.nio.file.Files;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.*;

/*
test methods
  positive
    installScoreAndCall
    updateScoreAndCall
  negative
    notEnoughBalance
    notEnoughStepLimit
    installWithInvalidParams
    updateWithInvalidParams
    updateWithInvalidOwner
    updateToInvalidScoreAddress
    invalidContentNoRootFile
    invalidContentNotZip
    invalidContentTooBig
    invalidScoreNoOnInstallMethod
    invalidScoreNoOnUpdateMethod
 */

@Tag(Constants.TAG_GOVERNANCE)
public class DeployTest {
    private static final String PACKAGE_JSON = "package.json";
    private static IconService iconService;
    private static Env.Chain chain;
    private static GovScore govScore;
    private static final BigInteger stepCostCC = BigInteger.valueOf(10);
    private static final BigInteger stepPrice = BigInteger.valueOf(1);
    private static final BigInteger invokeMaxStepLimit = BigInteger.valueOf(100000);
    private static GovScore.Fee fee;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        govScore = new GovScore(iconService, chain);
        fee = govScore.getFee();
        initDeploy();
    }

    public static void initDeploy() throws Exception {
        Utils.transferAndCheck(iconService, chain, chain.godWallet, chain.governorWallet.getAddress(), new BigInteger("10000000000"));
        govScore.setMaxStepLimit("invoke", invokeMaxStepLimit);
        govScore.setStepCost("contractCreate", stepCostCC);
        govScore.setStepPrice(stepPrice);
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    private Address deploy(KeyWallet owner, Address to, String contentPath, RpcObject params, long stepLimit) throws Exception {
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, owner, to, contentPath, params, stepLimit);
        return Utils.getAddressByTxHash(iconService, chain, txHash);
    }

    private void invoke(KeyWallet from, Address to, String method, RpcObject params) throws Exception {
        TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                .nid(BigInteger.valueOf(chain.networkId))
                .from(from.getAddress())
                .to(to)
                .stepLimit(BigInteger.valueOf(Constants.DEFAULT_STEP_LIMIT));

        Transaction t;
        if (params != null) {
            t = builder.call(method).params(params).build();
        } else {
            t = builder.call(method).build();
        }
        Bytes txHash = iconService.sendTransaction(new SignedTransaction(t, from)).execute();
        LOG.info("txHash [" + txHash + "]");
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }
    }

    @Test
    public void notEnoughBalance() throws Exception {
        LOG.infoEntering( "notEnoughBalance");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), BigInteger.valueOf(2));
        LOG.infoExiting();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        try {
            deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, 1);
        }
        // If StepTypeDefault or StepTypeInput is not 0, ResultTimeoutException will be happened.
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            LOG.info("FAIL to get result");
            return;
        }
        fail();
    }

    @Test
    public void notEnoughStepLimit() throws Exception {
        LOG.infoEntering( "notEnoughStepLimit");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), BigInteger.valueOf(1000));
        LOG.infoExiting();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        try {
            deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, 1);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            return;
        }
        fail();
    }

    @Test
    public void installWithInvalidParams() throws Exception {
        LOG.infoEntering( "installWithInvalidParams");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();
        RpcObject params = new RpcObject.Builder()
                .put("invalidParam", new RpcValue("invalid"))
                .build();
        try {
            deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, Constants.DEFAULT_STEP_LIMIT);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            return;
        }
        fail();
    }

    @Test
    public void updateWithInvalidParams() throws Exception {
        LOG.infoEntering( "updateWithInvalidParams");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        LOG.infoEntering("deploy");
        Address scoreAddr = deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, Constants.DEFAULT_STEP_LIMIT);
        LOG.infoExiting();

        LOG.infoEntering( "invoke");
        invoke(owner, scoreAddr, "hello", null);
        LOG.infoExiting();

        params = new RpcObject.Builder()
                .put("invalidParam", new RpcValue("invalid"))
                .build();
        try {
            LOG.infoEntering("update");
            deploy(owner, scoreAddr, Constants.SCORE_HELLOWORLD_UPDATE_PATH, params, Constants.DEFAULT_STEP_LIMIT);
            LOG.infoExiting();
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            LOG.infoExiting();
            return;
        }
        fail();
    }

    @Test
    public void installScoreAndCall() throws Exception {
        LOG.infoEntering( "installScoreAndCall");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        LOG.infoEntering("deploy");
        Address scoreAddr = deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, Constants.DEFAULT_STEP_LIMIT);
        LOG.infoExiting();

        LOG.infoEntering( "invoke");
        invoke(owner, scoreAddr, "hello", null);
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void updateScoreAndCall() throws Exception {
        LOG.infoEntering( "updateScoreAndCall");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();

        LOG.infoEntering("deploy");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Address scoreAddr = deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, Constants.DEFAULT_STEP_LIMIT);
        LOG.infoExiting();

        LOG.infoEntering( "invoke");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("ICONLOOP"))
                .build();
        invoke(owner, scoreAddr, "helloWithName", params);
        LOG.infoExiting();

        LOG.infoEntering("update");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        deploy(owner, scoreAddr, Constants.SCORE_HELLOWORLD_UPDATE_PATH, params, Constants.DEFAULT_STEP_LIMIT);
        LOG.infoExiting();

        LOG.infoEntering( "invoke");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("ICONLOOP"))
                .build();
        invoke(owner, scoreAddr, "helloWithName2", params);
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void updateWithInvalidOwner() throws Exception {
        LOG.infoEntering( "updateWithInvalidOwner");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();

        LOG.infoEntering("deploy");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Address scoreAddr = deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, Constants.DEFAULT_STEP_LIMIT);
        LOG.infoExiting();

        LOG.infoEntering( "invoke");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("ICONLOOP"))
                .build();
        invoke(owner, scoreAddr, "helloWithName", params);
        LOG.infoExiting();


        boolean failEx = false;
        LOG.infoExiting();
        try {
            LOG.infoEntering("update");
            params = new RpcObject.Builder()
                    .put("name", new RpcValue("HelloWorld"))
                    .build();
            deploy(KeyWallet.create(), scoreAddr, Constants.SCORE_HELLOWORLD_UPDATE_PATH, params, Constants.DEFAULT_STEP_LIMIT);
            LOG.infoExiting();
        }
        catch (ResultTimeoutException ex) {
            LOG.infoExiting();
            failEx = true;
        }
        assertTrue(failEx);

        LOG.infoEntering( "invoke not updated score method");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("ICONLOOP"))
                .build();
        invoke(owner, scoreAddr, "helloWithName", params);
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void updateToInvalidScoreAddress() throws Exception {
        LOG.infoEntering( "updateWithInvalidScoreAddress");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();

        LOG.infoEntering("deploy");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Address scoreAddr = deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, Constants.DEFAULT_STEP_LIMIT);
        LOG.infoExiting();

        LOG.infoEntering( "invoke");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("ICONLOOP"))
                .build();
        invoke(owner, scoreAddr, "helloWithName", params);
        LOG.infoExiting();


        boolean failEx = false;
        LOG.infoExiting();
        try {
            LOG.infoEntering("update");
            params = new RpcObject.Builder()
                    .put("name", new RpcValue("HelloWorld"))
                    .build();
            deploy(owner, KeyWallet.create().getAddress(), Constants.SCORE_HELLOWORLD_UPDATE_PATH, params, Constants.DEFAULT_STEP_LIMIT);
            LOG.infoExiting();
        }
        catch (TransactionFailureException ex) {
            LOG.infoExiting();
            failEx = true;
        }
        catch (ResultTimeoutException ex) {
            LOG.infoExiting();
            failEx = true;
        }
        assertTrue(failEx);

        LOG.infoEntering( "invoke not updated score method");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("ICONLOOP"))
                .build();
        invoke(owner, scoreAddr, "helloWithName", params);
        LOG.infoExiting();
        LOG.infoExiting();
    }

    private static void recursiveZip(File source, String zipPath, ZipOutputStream zos, boolean root) throws IOException {
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
            String path = zipPath == null ? dir : zipPath + dir;
            for(File file : files) {
                recursiveZip(file, path, zos, root);
            }
        }
        else {
            if (!root && source.getName().equals(PACKAGE_JSON)) {
                return;
            }
            ZipEntry ze = new ZipEntry(zipPath + source.getName());
            zos.putNextEntry(ze);
            zos.write(Files.readAllBytes(source.toPath()));
            zos.closeEntry();
        }
    }

    private Address installScore(byte []content, RpcObject params) throws Exception {
        LOG.infoEntering( "installScoreAndCall");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);
        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, owner.getAddress(), new BigInteger("10000000000000"));
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(new BigInteger("10000000000000"), bal);

        TransactionBuilder.DeployBuilder builder = TransactionBuilder.newBuilder()
                .nid(BigInteger.valueOf(chain.networkId))
                .from(owner.getAddress())
                .to(Constants.CHAINSCORE_ADDRESS)
                .stepLimit(BigInteger.valueOf(3000))
                .timestamp(Utils.getMicroTime())
                .nonce(new BigInteger("1"))
                .deploy(Constants.CONTENT_TYPE_PYTHON, content);
        if(params != null) {
            builder = builder.params(params);
        }
        Transaction transaction = builder.build();
        SignedTransaction signedTransaction = new SignedTransaction(transaction, owner);
        txHash = iconService.sendTransaction(signedTransaction).execute();
        result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            LOG.infoExiting();
            return null;
        }


        try {
            Utils.acceptIfAuditEnabled(iconService, chain, txHash);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            throw ex;
        }
        LOG.infoExiting();
        return new Address(result.getScoreAddress());
    }

    @Test
    public void invalidContentNoRootFile() throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        for(boolean includeRoot : new boolean[]{true, false}) {
            ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
            ZipOutputStream zos = new ZipOutputStream(outputStream);
            recursiveZip(new File(Constants.SCORE_HELLOWORLD_PATH), null, zos, includeRoot);
            zos.close();
            outputStream.close();
            byte[] content =  outputStream.toByteArray();
            try {
                assertEquals(includeRoot, installScore(content, params) != null);
            }
            catch (TransactionFailureException ex) {
                assertTrue(Utils.isAudit(iconService));
                assertFalse(includeRoot);
            }
        }
    }


    private static void readScore(File source, ByteArrayOutputStream bos) throws IOException {
        if(source.isHidden()) {
            return;
        }
        if(source.isDirectory()) {
            File []files = source.listFiles();
            for(File file : files) {
                readScore(file, bos);
            }
        }
        else {
            if (source.getName().equals(PACKAGE_JSON)) {
                return;
            }
            bos.write(Files.readAllBytes(source.toPath()));
        }
    }

    @Test
    public void invalidContentNotZip() throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        for(boolean zip : new boolean[]{true, false}) {
            ByteArrayOutputStream bos = new ByteArrayOutputStream();
            File source = new File(Constants.SCORE_HELLOWORLD_PATH);
            if(zip) {
                ZipOutputStream zos = new ZipOutputStream(bos);
                recursiveZip(new File(Constants.SCORE_HELLOWORLD_PATH), null, zos, true);
                zos.close();
                bos.close();
            }
            else {
                readScore(source, bos);
            }
            bos.close();
            byte[] content =  bos.toByteArray();
            try {
                assertEquals(zip, installScore(content, params) != null);
            }
            catch (TransactionFailureException ex) {
                assertTrue(Utils.isAudit(iconService));
                assertFalse(zip);
            }
        }
    }

    @Test
    public void invalidContentTooBig() throws Exception {
        String SCORE_TOO_BIG_PATH = Constants.SCORE_ROOT + "too_big";
        LOG.infoEntering( "invalidContentTooBig");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();

        LOG.infoEntering("deploy");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();

        try {
            deploy(owner, Constants.CHAINSCORE_ADDRESS, SCORE_TOO_BIG_PATH, params, Constants.DEFAULT_STEP_LIMIT);
            fail();
        }
        catch(Exception ex) {
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void invalidScoreNoOnInstallMethod() throws Exception {
        String SCORE_TOO_BIG_PATH = Constants.SCORE_ROOT + "no_install_method";
        LOG.infoEntering( "invalidScoreNoOnInstallMethod");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();

        LOG.infoEntering("deploy");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();

        try {
            deploy(owner, Constants.CHAINSCORE_ADDRESS, SCORE_TOO_BIG_PATH, params, Constants.DEFAULT_STEP_LIMIT);
            fail();
        }
        catch(TransactionFailureException ex) {
            LOG.info("FAIL to depoly : expected result");
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void invalidScoreNoOnUpdateMethod() throws Exception {
        LOG.infoEntering( "invalidScoreNoOnUpdateMethod");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        LOG.infoEntering("transfer and check balance");
        Utils.transferAndCheck(iconService, chain, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        LOG.infoExiting();

        LOG.infoEntering("deploy");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Address scoreAddr = deploy(owner, Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, Constants.DEFAULT_STEP_LIMIT);
        LOG.infoExiting();

        LOG.infoEntering( "invoke");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("ICONLOOP"))
                .build();
        invoke(owner, scoreAddr, "helloWithName", params);
        LOG.infoExiting();

        boolean failEx = false;
        String noUpdatePath = Constants.SCORE_ROOT + "no_update_method";
        try {
            LOG.infoEntering("update with no update method");
            deploy(owner, scoreAddr, noUpdatePath, null, Constants.DEFAULT_STEP_LIMIT);
            LOG.infoExiting();
        }
        catch (TransactionFailureException ex) {
            LOG.infoExiting();
            failEx = true;
        }
        assertTrue(failEx);

        LOG.infoEntering( "invoke not updated score method");
        params = new RpcObject.Builder()
                .put("name", new RpcValue("ICONLOOP"))
                .build();
        invoke(owner, scoreAddr, "helloWithName", params);
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void invalidSignature() throws Exception {
        LOG.infoEntering( "invalidSignature");
        KeyWallet testWallets[] = new KeyWallet[10];
        Address testAddr[] = new Address[10];
        LOG.infoEntering( "transfer for test");

        for(int i = 0; i < testWallets.length; i++) {
            testWallets[i] = KeyWallet.create();
            testAddr[i] = testWallets[i].getAddress();
        }
        Utils.transferAndCheck(iconService, chain, chain.godWallet, testAddr, Constants.DEFAULT_BALANCE);
        LOG.infoExiting();
        byte[] content = Utils.zipContent(Constants.SCORE_HELLOWORLD_PATH);
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        for(int i = 0; i < testWallets.length; i++) { //            Transaction transaction = TransactionBuilder.newBuilder()
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(BigInteger.valueOf(chain.networkId))
                    .from(testWallets[i].getAddress())
                    .to(Constants.CHAINSCORE_ADDRESS)
                    .stepLimit(BigInteger.valueOf(Constants.DEFAULT_STEP_LIMIT))
                    .timestamp(Utils.getMicroTime())
                    .nonce(new BigInteger("1"))
                    .deploy(Constants.CONTENT_TYPE_PYTHON, content)
                    .params(params)
                    .build();

            SignedTransaction signedTransaction = new SignedTransaction(transaction, testWallets[0]);
            try {
                Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
                assertEquals(0, i);
                Address addr = Utils.getAddressByTxHash(iconService, chain, txHash);
                invoke(testWallets[0], addr, "hello", null);
            }
            catch(RpcError ex) {
                assertNotEquals(0, i);
                continue;
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void deployGovScore() throws Exception {
        LOG.infoEntering("setGovernance");
        final String guPath = Constants.SCORE_GOV_UPDATE_PATH;
        // deploy tx to governance address
        boolean updated = Utils.icxCall(iconService, Constants.GOV_ADDRESS,
                "updated",null).asBoolean();
        assertTrue(!updated);

        // Update with not owner
        LOG.infoEntering("update governance score with not governor");
        RpcObject govParams = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .put("value", new RpcValue("0x1"))
                .build();
        Bytes txHash = Utils.deployScore(iconService, chain.networkId,
                chain.godWallet, Constants.GOV_ADDRESS, guPath, govParams);
        TransactionResult result = Utils.getTransactionResult(iconService,
                txHash, Constants.DEFAULT_WAITING_TIME);
        LOG.infoExiting("result : " + result);
        assertEquals(Constants.STATUS_FAIL, result.getStatus());

        // Update with governor
        LOG.infoEntering("update governance score with governor");
        KeyWallet govWallet = chain.governorWallet;
        txHash = Utils.deployScore(iconService, chain.networkId,
                govWallet, Constants.GOV_ADDRESS, guPath, null);
        result = Utils.getTransactionResult(iconService,
                txHash, Constants.DEFAULT_WAITING_TIME);

        try {
            Utils.acceptIfAuditEnabled(iconService, chain, txHash);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            throw ex;
        }
        LOG.infoExiting("result : " + result);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        LOG.info("result : " + result);
        updated = Utils.icxCall(iconService, Constants.GOV_ADDRESS, "updated",null).asBoolean();
        assertTrue(updated);
        LOG.infoExiting();
    }
}
