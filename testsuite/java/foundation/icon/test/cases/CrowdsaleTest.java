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
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.CrowdSaleScore;
import foundation.icon.test.score.SampleTokenScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

class CrowdsaleTest extends TestBase {
    private static TransactionHandler txHandler;
    private static KeyWallet[] wallets;
    private static KeyWallet ownerWallet;

    @BeforeAll
    static void setup() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);

        // init wallets
        wallets = new KeyWallet[3];
        BigInteger amount = ICX.multiply(BigInteger.valueOf(30));
        for (int i = 0; i < wallets.length; i++) {
            wallets[i] = KeyWallet.create();
            txHandler.transfer(wallets[i].getAddress(), amount);
        }
        for (KeyWallet wallet : wallets) {
            ensureIcxBalance(txHandler, wallet.getAddress(), BigInteger.ZERO, amount);
        }
        ownerWallet = wallets[0];
    }

    @AfterAll
    static void shutdown() throws Exception {
        for (KeyWallet wallet : wallets) {
            txHandler.refundAll(wallet);
        }
    }

    @Tag(Constants.TAG_PY_SCORE)
    @Test
    void testPythonToPython() throws Exception {
        deployAndStartCrowdsale(Constants.CONTENT_TYPE_PYTHON, Constants.CONTENT_TYPE_PYTHON);
    }

    @Tag(Constants.TAG_JAVA_SCORE)
    @Test
    void testJavaToJava() throws Exception {
        deployAndStartCrowdsale(Constants.CONTENT_TYPE_JAVA, Constants.CONTENT_TYPE_JAVA);
    }

    @Tag(Constants.TAG_JAVA_GOV)
    @Test
    void testPythonToJava() throws Exception {
        deployAndStartCrowdsale(Constants.CONTENT_TYPE_PYTHON, Constants.CONTENT_TYPE_JAVA);
    }

    @Tag(Constants.TAG_JAVA_GOV)
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
        KeyWallet aliceWallet = wallets[1];
        KeyWallet bobWallet = wallets[2];

        // send 50 icx to Alice, 100 to Bob
        LOG.infoEntering("transfer icx", "50 to Alice; 100 to Bob");
        transferAndCheckResult(txHandler, aliceWallet.getAddress(), ICX.multiply(BigInteger.valueOf(50)));
        transferAndCheckResult(txHandler, bobWallet.getAddress(), ICX.multiply(BigInteger.valueOf(100)));
        LOG.infoExiting();

        // transfer all tokens to crowdsale score
        LOG.infoEntering("transfer token", "all tokens to crowdsale score from owner");
        TransactionResult result = tokenScore.transfer(ownerWallet, crowdsaleScore.getAddress(), ICX.multiply(initialSupply));
        crowdsaleScore.ensureFundingGoal(result, fundingGoalInIcx);
        tokenScore.ensureTokenBalance(crowdsaleScore.getAddress(), initialSupply.longValue());
        LOG.infoExiting();

        // send icx to crowdsale score from Alice and Bob
        LOG.infoEntering("transfer icx", "to crowdsale score (40 from Alice, 60 from Bob)");
        Bytes[] ids = new Bytes[2];
        ids[0] = txHandler.transfer(aliceWallet, crowdsaleScore.getAddress(), ICX.multiply(BigInteger.valueOf(40)));
        ids[1] = txHandler.transfer(bobWallet, crowdsaleScore.getAddress(), ICX.multiply(BigInteger.valueOf(60)));
        for (Bytes id : ids) {
            assertSuccess(txHandler.getResult(id));
        }
        tokenScore.ensureTokenBalance(aliceWallet.getAddress(), 40);
        tokenScore.ensureTokenBalance(bobWallet.getAddress(), 60);
        LOG.infoExiting();

        // check if goal reached
        LOG.infoEntering("call", "checkGoalReached()");
        crowdsaleScore.ensureCheckGoalReached(ownerWallet);
        LOG.infoExiting();

        // do safe withdrawal
        LOG.infoEntering("call", "safeWithdrawal()");
        BigInteger oldBal = txHandler.getBalance(ownerWallet.getAddress());
        result = crowdsaleScore.safeWithdrawal(ownerWallet);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new IOException("Failed to execute safeWithdrawal.");
        }
        BigInteger amount = ICX.multiply(BigInteger.valueOf(100));
        crowdsaleScore.ensureFundTransfer(result, ownerWallet.getAddress(), amount);

        // check the final icx balance of owner
        LOG.info("ICX balance before safeWithdrawal: " + oldBal);
        BigInteger fee = result.getStepUsed().multiply(result.getStepPrice());
        BigInteger newBal = oldBal.add(amount).subtract(fee);
        ensureIcxBalance(txHandler, ownerWallet.getAddress(), oldBal, newBal);
        LOG.infoExiting();
    }
}
