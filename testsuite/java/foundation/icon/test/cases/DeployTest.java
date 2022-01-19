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
import foundation.icon.icx.crypto.IconKeys;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.HelloWorld;
import foundation.icon.test.score.Score;
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
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.fail;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

@Tag(Constants.TAG_PY_GOV)
public class DeployTest extends TestBase {
    private static final String PACKAGE_JSON = "package.json";
    private static final BigInteger stepCostCC = BigInteger.valueOf(10);
    private static final BigInteger stepPrice = BigInteger.valueOf(1);
    private static final BigInteger invokeMaxStepLimit = BigInteger.valueOf(10000000);
    private static final BigInteger stepsForDeploy = stepCostCC;

    private static TransactionHandler txHandler;
    private static Env.Chain chain;
    private static ChainScore chainScore;
    private static GovScore govScore;
    private static GovScore.Fee fee;
    private static KeyWallet[] testWallets;
    private static final int testWalletNum = 2;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        chainScore = new ChainScore(txHandler);
        govScore = new GovScore(txHandler);
        fee = govScore.getFee();
        initDeploy();
    }

    public static void initDeploy() throws Exception {
        testWallets = new KeyWallet[testWalletNum];
        Address[] testAddresses = new Address[testWalletNum];
        for (int i = 0; i < testWalletNum; i++) {
            testWallets[i] = KeyWallet.create();
            testAddresses[i] = testWallets[i].getAddress();
        }
        transferAndCheckResult(txHandler, chain.governorWallet.getAddress(), ICX);
        transferAndCheckResult(txHandler, testAddresses, ICX);
        govScore.setMaxStepLimit("invoke", invokeMaxStepLimit);
        govScore.setStepCost("contractCreate", stepCostCC);
        govScore.setStepPrice(stepPrice);
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    private Score deployHello(KeyWallet owner, String scorePath, RpcObject params, BigInteger stepLimit)
            throws IOException, ResultTimeoutException, TransactionFailureException {
        if (scorePath == null) {
            scorePath = HelloWorld.INSTALL_PATH;
        }
        if (params == null) {
            params = new RpcObject.Builder()
                    .put("name", new RpcValue("HelloWorld"))
                    .build();
        }
        return txHandler.deploy(owner, scorePath, params, stepLimit);
    }

    private Score updateHello(Address to, KeyWallet owner, String scorePath, RpcObject params)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        if (scorePath == null) {
            scorePath = HelloWorld.UPDATE_PATH;
        }
        if (params == null) {
            params = new RpcObject.Builder()
                    .put("name", new RpcValue("Updated HelloWorld"))
                    .build();
        }
        return txHandler.deploy(owner, scorePath, to, params, stepsForDeploy);
    }

    @Test
    public void notEnoughBalance() throws Exception {
        LOG.infoEntering("notEnoughBalance");
        KeyWallet owner = KeyWallet.create();
        transferAndCheckResult(txHandler, owner.getAddress(), stepsForDeploy.subtract(BigInteger.ONE));
        try {
            deployHello(owner, null, null, stepsForDeploy);
            fail();
        } catch (ResultTimeoutException e) {
            LOG.info("Expected exception: msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
    }

    @Test
    public void notEnoughStepLimit() throws Exception {
        LOG.infoEntering("notEnoughStepLimit");
        KeyWallet owner = KeyWallet.create();
        transferAndCheckResult(txHandler, owner.getAddress(), stepsForDeploy);
        try {
            deployHello(owner, null, null, stepsForDeploy.subtract(BigInteger.ONE));
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
    }

    @Test
    public void installWithInvalidParams() {
        LOG.infoEntering("installWithInvalidParams");
        KeyWallet owner = testWallets[0];
        try {
            RpcObject params = new RpcObject.Builder()
                    .put("invalidParam", new RpcValue("invalid"))
                    .build();
            deployHello(owner, null, params, stepsForDeploy);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
    }

    @Test
    public void updateTests() throws Exception {
        LOG.infoEntering("updateTests");
        KeyWallet owner = testWallets[0];
        Score helloScore = deployHello(owner, null, null, stepsForDeploy);

        LOG.infoEntering("invoke", "hello");
        helloScore.invokeAndWaitResult(owner, "hello", null);
        LOG.infoExiting();

        LOG.infoEntering("update", "with invalid params");
        try {
            RpcObject params = new RpcObject.Builder()
                    .put("invalidParam", new RpcValue("invalid"))
                    .build();
            updateHello(helloScore.getAddress(), owner, null, params);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }

        LOG.infoEntering("update", "with invalid owner");
        try {
            KeyWallet another = testWallets[1];
            updateHello(helloScore.getAddress(), another, null, null);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }

        LOG.infoEntering("update", "to invalid address");
        try {
            Address invalid = new Address(Address.AddressPrefix.CONTRACT,
                                          IconKeys.getAddressHash(KeyWallet.create().getPublicKey().toByteArray()));
            updateHello(invalid, owner, null, null);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }

        LOG.infoEntering("update", "success case");
        try {
            helloScore = updateHello(helloScore.getAddress(), owner, null, null);
            RpcObject params = new RpcObject.Builder()
                    .put("name", new RpcValue("Alice"))
                    .build();
            assertSuccess(helloScore.invokeAndWaitResult(owner, "helloWithName2", params));
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    private static void recursiveZip(File source, String zipPath, ZipOutputStream zos, boolean includeRoot) throws IOException {
        if (source.isHidden()) {
            return;
        }
        if (source.isDirectory()) {
            String dir = source.getName();
            if (!dir.endsWith(File.separator)) {
                dir = dir + File.separator;
            }
            zos.putNextEntry(new ZipEntry(dir));
            zos.closeEntry();
            File[] files = source.listFiles();
            String path = (zipPath == null) ? dir : zipPath + dir;
            for (File file : files) {
                recursiveZip(file, path, zos, includeRoot);
            }
        } else {
            if (!includeRoot && source.getName().equals(PACKAGE_JSON)) {
                return;
            }
            ZipEntry ze = new ZipEntry(zipPath + source.getName());
            zos.putNextEntry(ze);
            zos.write(Files.readAllBytes(source.toPath()));
            zos.closeEntry();
        }
    }

    private Address installScore(byte[] content) throws Exception {
        LOG.infoEntering("installScore");
        KeyWallet owner = testWallets[0];
        try {
            RpcObject params = new RpcObject.Builder()
                    .put("name", new RpcValue("HelloWorld"))
                    .build();
            Score score = txHandler.deploy(owner, content, params, Constants.DEFAULT_STEPS);
            return score.getAddress();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
            return null;
        } finally {
            LOG.infoExiting();
        }
    }

    @Test
    public void invalidContentNoRootFile() throws Exception {
        for (boolean includeRoot : new boolean[]{true, false}) {
            LOG.infoEntering("includeRoot", String.valueOf(includeRoot));
            ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
            ZipOutputStream zos = new ZipOutputStream(outputStream);
            recursiveZip(new File(HelloWorld.INSTALL_PATH), null, zos, includeRoot);
            zos.close();
            outputStream.close();
            byte[] content = outputStream.toByteArray();
            assertEquals(includeRoot, installScore(content) != null);
            LOG.infoExiting();
        }
    }

    private static void readScore(File source, ByteArrayOutputStream bos) throws IOException {
        if (source.isHidden()) {
            return;
        }
        if (source.isDirectory()) {
            File[] files = source.listFiles();
            for (File file : files) {
                readScore(file, bos);
            }
        } else {
            if (source.getName().equals(PACKAGE_JSON)) {
                return;
            }
            bos.write(Files.readAllBytes(source.toPath()));
        }
    }

    @Test
    public void invalidContentNotZip() throws Exception {
        for (boolean zip : new boolean[]{true, false}) {
            LOG.infoEntering("validZip", String.valueOf(zip));
            ByteArrayOutputStream bos = new ByteArrayOutputStream();
            File source = new File(HelloWorld.INSTALL_PATH);
            if (zip) {
                ZipOutputStream zos = new ZipOutputStream(bos);
                recursiveZip(source, null, zos, true);
                zos.close();
                bos.close();
            } else {
                readScore(source, bos);
            }
            bos.close();
            byte[] content = bos.toByteArray();
            assertEquals(zip, installScore(content) != null);
            LOG.infoExiting();
        }
    }

    @Test
    public void invalidContentTooBig() {
        LOG.infoEntering("invalidContentTooBig");
        KeyWallet owner = testWallets[0];
        try {
            RpcObject params = new RpcObject.Builder()
                    .put("name", new RpcValue("HelloWorld"))
                    .build();
            txHandler.deploy(owner, Score.getFilePath("too_big"), params, stepsForDeploy);
            fail();
        } catch (RpcError e) {
            LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
    }

    @Test
    public void invalidScoreNoOnInstallMethod() {
        LOG.infoEntering("invalidScoreNoOnInstallMethod");
        KeyWallet owner = testWallets[0];
        try {
            deployHello(owner, Score.getFilePath("no_install_method"), null, stepsForDeploy);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
    }

    @Test
    public void invalidScoreNoOnUpdateMethod() throws Exception {
        LOG.infoEntering("invalidScoreNoOnUpdateMethod");
        KeyWallet owner = testWallets[0];
        Score helloScore = deployHello(owner, null, null, stepsForDeploy);

        LOG.infoEntering("invoke", "hello");
        helloScore.invokeAndWaitResult(owner, "hello", null);
        LOG.infoExiting();

        LOG.infoEntering("update", "without on_update method");
        try {
            updateHello(helloScore.getAddress(), owner, Score.getFilePath("no_update_method"), null);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void deployGovScore() throws Exception {
        LOG.infoEntering("setGovernance");
        // check the existing governance score
        boolean updated = govScore.call("updated", null).asBoolean();
        assertFalse(updated);

        // Update with invalid governor
        LOG.infoEntering("update governance", "with invalid governor");
        KeyWallet testWallet = testWallets[0];
        try {
            txHandler.deploy(testWallet, GovScore.UPDATE_PATH,
                    Constants.GOV_ADDRESS, null, Constants.DEFAULT_STEPS);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }

        // Update with governor
        LOG.infoEntering("update governance", "with governor");
        try {
            Score updatedGov = txHandler.deploy(chain.governorWallet, GovScore.UPDATE_PATH,
                    Constants.GOV_ADDRESS, null, Constants.DEFAULT_STEPS);
            updated = updatedGov.call("updated", null).asBoolean();
            assertTrue(updated);
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void testDeployerWhiteList() throws Exception {
        assumeTrue(chainScore.isDeployerWhiteListEnabled(), "deployerWhiteList is not enabled.");
        LOG.infoEntering("setup", "test wallets");
        KeyWallet deployer = testWallets[0];
        KeyWallet caller = testWallets[1];
        LOG.infoExiting();

        LOG.infoEntering("invoke", "addDeployer");
        TransactionResult result = govScore.addDeployer(deployer.getAddress());
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("deploy", "by deployer");
        HelloWorld helloScore = HelloWorld.install(txHandler, deployer);
        assertEquals(Constants.STATUS_SUCCESS, helloScore.invokeHello(caller).getStatus());
        LOG.infoExiting();

        LOG.infoEntering("deploy", "by caller");
        try {
            HelloWorld.install(txHandler, caller);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        }
        LOG.infoExiting();
    }

    @Test
    public void testDeployMultipleWithoutAccept() throws Exception {
        assumeTrue(chainScore.isAuditEnabled(), "audit is not enabled");

        LOG.infoEntering("testDeployMultipleWithoutAccept");

        KeyWallet owner = testWallets[0];
        Address score = null;
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();

        LOG.infoEntering("inital deploy");
        try {
            var txHash = txHandler.deployOnly(owner, HelloWorld.INSTALL_PATH, params);
            var result = txHandler.getResult(txHash);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            score = new Address(result.getScoreAddress());
        } catch (Exception e) {
            fail(e);
            return;
        }
        LOG.infoExiting();

        LOG.infoEntering("deploy again");
        try {
            var txHash = txHandler.deployOnly(owner, score, HelloWorld.INSTALL_PATH, params);
            var result = txHandler.getResult(txHash);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        } catch (Exception e) {
            fail(e);
            return;
        }
        LOG.infoExiting();

        LOG.infoEntering("deploy again and accept");
        try {
            var txHash = txHandler.deployOnly(owner, score, HelloWorld.INSTALL_PATH, params);
            var result = txHandler.getResult(txHash);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

            txHandler.acceptScoreIfAuditEnabled(txHash);
        } catch (Exception e) {
            fail(e);
            return;
        }
        LOG.infoExiting();

        LOG.infoExiting();
    }
}
