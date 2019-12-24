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
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.CrowdSaleScore;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.SampleTokenScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

class CrowdsaleTest {
    private static IconService iconService;
    private static TransactionHandler txHandler;
    private static Env.Chain chain;
    private static KeyWallet ownerWallet;
    private static GovScore govScore;
    private static GovScore.Fee fee;

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        govScore = new GovScore(iconService, chain);
        fee = govScore.getFee();
        initScoreTest();
    }

    private static void initScoreTest() throws Exception {
        ownerWallet = KeyWallet.create();
        Utils.transferAndCheck(iconService, chain, chain.godWallet, new Address[] {
                    ownerWallet.getAddress(), chain.governorWallet.getAddress()
                }, BigInteger.TEN.pow(20));

        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000000));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(1000000));
    }

    @AfterAll
    static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    @Tag(Constants.TAG_GOVERNANCE)
    @Tag(Constants.TAG_JAVA_SCORE)
    @Test
    void testPythonToPython() throws Exception {
        deployAndStartCrowdsale(Constants.CONTENT_TYPE_PYTHON, Constants.CONTENT_TYPE_PYTHON);
    }

    @Tag(Constants.TAG_JAVA_SCORE)
    @Test
    void testJavaToJava() throws Exception {
        deployAndStartCrowdsale(Constants.CONTENT_TYPE_JAVA, Constants.CONTENT_TYPE_JAVA);
    }

    @Tag(Constants.TAG_JAVA_SCORE)
    @Test
    void testPythonToJava() throws Exception {
        deployAndStartCrowdsale(Constants.CONTENT_TYPE_PYTHON, Constants.CONTENT_TYPE_JAVA);
    }

    @Tag(Constants.TAG_JAVA_SCORE)
    @Test
    void testJavaToPython() throws Exception {
        deployAndStartCrowdsale(Constants.CONTENT_TYPE_JAVA, Constants.CONTENT_TYPE_PYTHON);
    }

    void deployAndStartCrowdsale(String tokenType, String crowdsaleType) throws Exception {
        // deploy token SCORE
        BigInteger decimals = BigInteger.valueOf(18);
        BigInteger initialSupply = BigInteger.valueOf(1000);
        SampleTokenScore tokenScore = SampleTokenScore.mustDeploy(txHandler, ownerWallet,
                decimals, initialSupply, tokenType);

        // deploy crowdsale SCORE
        BigInteger fundingGoalInIcx = BigInteger.valueOf(100);
        CrowdSaleScore crowdsaleScore = CrowdSaleScore.mustDeploy(txHandler, ownerWallet,
                tokenScore.getAddress(), fundingGoalInIcx, crowdsaleType);

        startCrowdsale(tokenScore, crowdsaleScore, initialSupply, fundingGoalInIcx);
    }

    void startCrowdsale(SampleTokenScore tokenScore, CrowdSaleScore crowdsaleScore,
                        BigInteger initialSupply, BigInteger fundingGoalInIcx) throws Exception {
        KeyWallet aliceWallet = KeyWallet.create();
        KeyWallet bobWallet = KeyWallet.create();
        BigInteger ownerBalance = iconService.getBalance(ownerWallet.getAddress()).execute();

        // send 50 icx to Alice, 100 to Bob
        LOG.infoEntering("transfer icx", "50 to Alice; 100 to Bob");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, aliceWallet.getAddress(), "50");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, bobWallet.getAddress(), "100");
        Utils.ensureIcxBalance(iconService, aliceWallet.getAddress(), 0, 50);
        Utils.ensureIcxBalance(iconService, bobWallet.getAddress(), 0, 100);
        LOG.infoExiting();

        // transfer all tokens to crowdsale score
        LOG.infoEntering("transfer token", "all tokens to crowdsale score from owner");
        Bytes txHash = tokenScore.transfer(ownerWallet, crowdsaleScore.getAddress(), initialSupply);
        crowdsaleScore.ensureFundingGoal(txHash, fundingGoalInIcx);
        tokenScore.ensureTokenBalance(crowdsaleScore.getAddress(), initialSupply.longValue());
        LOG.infoExiting();

        // send icx to crowdsale score from Alice and Bob
        LOG.infoEntering("transfer icx", "to crowdsale score (40 from Alice, 60 from Bob)");
        Utils.transferIcx(iconService, chain.networkId, aliceWallet, crowdsaleScore.getAddress(), "40");
        Utils.transferIcx(iconService, chain.networkId, bobWallet, crowdsaleScore.getAddress(), "60");
        tokenScore.ensureTokenBalance(aliceWallet.getAddress(), 40);
        tokenScore.ensureTokenBalance(bobWallet.getAddress(), 60);
        LOG.infoExiting();

        // check if goal reached
        LOG.infoEntering("call", "checkGoalReached()");
        crowdsaleScore.ensureCheckGoalReached(ownerWallet);
        LOG.infoExiting();

        // do safe withdrawal
        LOG.infoEntering("call", "safeWithdrawal()");
        TransactionResult result = crowdsaleScore.safeWithdrawal(ownerWallet);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new IOException("Failed to execute safeWithdrawal.");
        }
        BigInteger amount = IconAmount.of("100", IconAmount.Unit.ICX).toLoop();
        tokenScore.ensureFundTransfer(result, crowdsaleScore.getAddress(), ownerWallet.getAddress(), amount);

        // check the final icx balance of owner
        LOG.info("Initial ICX balance of owner: " + ownerBalance);
        Utils.ensureIcxBalance(iconService, ownerWallet.getAddress(), ownerBalance, ownerBalance.add(amount));
        LOG.infoExiting();
    }
}
