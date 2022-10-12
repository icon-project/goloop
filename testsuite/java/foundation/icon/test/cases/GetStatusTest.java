/*
 * Copyright 2022 ICON Foundation
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

import foundation.icon.ee.util.Crypto;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.ScoreStatus;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.common.ZipFile;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import testcases.APITest;

import java.math.BigInteger;
import java.security.SecureRandom;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.*;

public class GetStatusTest extends TestBase {
    private static IconService iconService;
    private static TransactionHandler txHandler;
    private static KeyWallet[] wallets;
    private static KeyWallet ownerWallet, caller;
    private static GovScore govScore;
    private static ChainScore chainScore;

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        govScore = new GovScore(txHandler);
        chainScore = new ChainScore(txHandler);

        // init wallets
        wallets = new KeyWallet[2];
        BigInteger amount = ICX.multiply(BigInteger.valueOf(200));
        for (int i = 0; i < wallets.length; i++) {
            wallets[i] = KeyWallet.create();
            txHandler.transfer(wallets[i].getAddress(), amount);
        }
        for (KeyWallet wallet : wallets) {
            ensureIcxBalance(txHandler, wallet.getAddress(), BigInteger.ZERO, amount);
        }
        ownerWallet = wallets[0];
        caller = wallets[1];
    }

    @AfterAll
    static void shutdown() throws Exception {
        for (KeyWallet wallet : wallets) {
            txHandler.refundAll(wallet);
        }
    }

    public class DeployInfo {
        public final byte[] content;
        public final Bytes deployTx;
        public final Bytes auditTx;
        public final Score score;

        public DeployInfo(byte[] content, Bytes deployTx, Bytes auditTx, Score score) {
            this.content = content;
            this.deployTx = deployTx;
            this.auditTx = auditTx;
            this.score = score;
        }
    }

    private DeployInfo deployJavaScore() throws Exception {
        LOG.infoEntering("deploy", "APITest");
        var deployContent = txHandler.makeJar(APITest.class.getName(), new Class<?>[]{ APITest.class } );
        var deployTx = txHandler.deployOnly(ownerWallet, Constants.CHAINSCORE_ADDRESS, deployContent, null, Constants.CONTENT_TYPE_JAVA);
        var score = txHandler.getScore(deployTx, true);
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new DeployInfo(deployContent, deployTx, deployTx, score);
    }

    private DeployInfo deployPythonScore() throws Exception {
        LOG.infoEntering("deploy", "score_api");
        var deployContent = ZipFile.zipContent(Score.getFilePath("score_api"));
        var deployTx = txHandler.deployOnly(ownerWallet, Constants.CHAINSCORE_ADDRESS, deployContent, null, Constants.CONTENT_TYPE_PYTHON);
        var result = txHandler.getResult(deployTx);
        assertStatus(Constants.STATUS_SUCCESS, result);
        var scoreAddress = result.getScoreAddress();
        var auditTx = deployTx;
        if (govScore.isAuditEnabledOnly()) {
            result = govScore.acceptScore(deployTx);
            assertStatus(Constants.STATUS_SUCCESS, result);
            auditTx = result.getTxHash();
        }
        var score = new Score(txHandler, new Address(scoreAddress));
        LOG.infoExiting();
        return new DeployInfo(deployContent, deployTx, auditTx, score);
    }

    private ScoreStatus getScoreStatus(Address address) throws Exception {
        LOG.infoEntering("getScoreStatus", "address="+address);
        try {
            var req = iconService.getScoreStatus(address);
            return req.execute();
        } finally {
            LOG.infoExiting();
        }
    }

    @Test
    @Tag(Constants.TAG_JAVA_SCORE)
    public void testScoreStatusForJavaScore() throws Exception {
        LOG.infoEntering("testScoreStatusForJavaScore");
        var deploy = deployJavaScore();
        testScoreStatusFor(deploy);
        LOG.infoExiting();
    }

    @Test
    @Tag(Constants.TAG_PY_SCORE)
    @Tag(Constants.TAG_PY_GOV)
    public void testScoreStatusForPythonScore() throws Exception {
        LOG.infoEntering("testScoreStatusForJavaScore");
        var deploy = deployPythonScore();
        testScoreStatusFor(deploy);
        LOG.infoExiting();
    }

    private void testScoreStatusFor(DeployInfo deploy) throws Exception {
        LOG.infoEntering("testScoreStatusFor");

        LOG.infoEntering("icx_getScoreStatus with valid address");
        var contentHash = new Bytes(Crypto.sha3_256(deploy.content));
        var scoreAddress = deploy.score.getAddress();
        var status = getScoreStatus(scoreAddress);
        LOG.info("Status = "+status);
        assertEquals(ownerWallet.getAddress(), status.getOwner());
        var current = status.getCurrent();
        assertEquals(deploy.deployTx, current.getDeployTxHash());
        assertEquals(deploy.auditTx, current.getAuditTxHash());
        assertEquals(contentHash, current.getCodeHash());
        LOG.infoExiting();

        LOG.infoEntering("icx_getScoreStatus with changed prefix");

        var eoaAddress = new Address(Address.AddressPrefix.EOA, scoreAddress.getBody());

        try {
            getScoreStatus(eoaAddress);
            fail("Unexpected success");
        } catch (RpcError e) {
            LOG.info("Expected failure="+e);
        }

        LOG.infoExiting();
    }

    @Test
    @Tag(Constants.TAG_JAVA_SCORE)
    public void testScoreStatusOnChainScore() throws Exception {
        LOG.infoEntering("testScoreStatusOnChainScore");
        var address = new Address("cx0000000000000000000000000000000000000000");

        var result = getScoreStatus(address);
        assertNull(result.getOwner(), "Owner must be null");
        var contract = result.getCurrent();
        assertNull(contract.getDeployTxHash());
        assertNull(contract.getAuditTxHash());

        LOG.infoExiting();
    }

    @Test
    @Tag(Constants.TAG_JAVA_SCORE)
    public void testScoreStatusWithEoA() throws Exception {
        LOG.infoEntering("testScoreStatusWithEoA");
        try {
            getScoreStatus(caller.getAddress());
        } catch (RpcError e) {
            LOG.info("Expected failure:"+e);
        }
        LOG.infoExiting();
    }

    @Test
    @Tag(Constants.TAG_JAVA_SCORE)
    public void testScoreStatusWithFakedContract() throws Exception {
        LOG.infoEntering("testScoreStatusWithFakedContract");
        var eoa = caller.getAddress();
        var faked = new Address(Address.AddressPrefix.CONTRACT, eoa.getBody());
        try {
            getScoreStatus(faked);
        } catch (RpcError e) {
            LOG.info("Expected failure:"+e);
        }
        LOG.infoExiting();
    }

    @Test
    @Tag(Constants.TAG_JAVA_GOV)
    public void testScoreStatusOnGovernanceWithJava() throws Exception {
        LOG.infoEntering("testScoreStatusOnGovernanceWithJava");
        var deployInfo = deployJavaScore();
        testScoreStatusOnGovernanceWith(deployInfo.score);
        LOG.infoExiting();
    }

    @Test
    @Tag(Constants.TAG_PY_GOV)
    public void testScoreStatusOnGovernanceWithPython() throws Exception {
        LOG.infoEntering("testScoreStatusOnGovernanceWithPython");
        var deployInfo = deployPythonScore();
        testScoreStatusOnGovernanceWith(deployInfo.score);
        LOG.infoExiting();
    }

    private void testScoreStatusOnGovernanceWith(Score score) throws Exception {
        LOG.infoEntering("testScoreStatusOnGovernanceWith");

        LOG.info("Ensure no flags are set");
        var status = getScoreStatus(score.getAddress());
        assertFalse(status.isDisabled());
        assertFalse(status.isBlocked());
        assertFalse(status.useSystemDeposit());

        RpcObject params;
        TransactionResult result;

        LOG.infoEntering("Disabled status test");

        LOG.infoEntering("Disable the score", "score="+score.getAddress());
        params = new RpcObject.Builder()
                .put("address", new RpcValue(score.getAddress()))
                .build();
        result = chainScore.invokeAndWaitResult(ownerWallet, "disableScore", params);
        assertStatus(Constants.STATUS_SUCCESS, result);
        LOG.infoExiting();

        status = getScoreStatus(score.getAddress());

        LOG.info("Ensure it's disabled");
        assertTrue(status.isDisabled());

        LOG.infoExiting();

        LOG.infoEntering("Blocked status test");

        LOG.infoEntering("Block the score", "score="+score.getAddress());
        params = new RpcObject.Builder()
                .put("address", new RpcValue(score.getAddress()))
                .build();
        result = govScore.invokeAndWaitResult(txHandler.getChain().governorWallet,
                "blockScore", params);
        assertStatus(Constants.STATUS_SUCCESS, result);
        LOG.infoExiting();

        status = getScoreStatus(score.getAddress());
        LOG.info("Ensure it's blocked");
        assertTrue(status.isBlocked());

        LOG.infoExiting();

        var rev = chainScore.getRevision();
        if (rev >= 9) {
            LOG.infoEntering("useSystemDeposit");

            LOG.infoEntering("set useSystemDeposit", "score="+score.getAddress());
            params = new RpcObject.Builder()
                    .put("address", new RpcValue(score.getAddress()))
                    .put("yn", new RpcValue(true))
                    .build();
            result = govScore.invokeAndWaitResult(txHandler.getChain().governorWallet,
                    "setUseSystemDeposit", params);
            assertStatus(Constants.STATUS_SUCCESS, result);
            LOG.infoExiting();

            status = getScoreStatus(score.getAddress());
            LOG.info("Ensure that it uses system deposit");
            assertTrue(status.useSystemDeposit());

            LOG.infoExiting();
        } else {
            LOG.info("Skipping useSystemDeposit rev="+rev+" (>=9 required)");
        }

        LOG.infoExiting();
    }
}
