/*
 * Copyright 2020 ICON Foundation
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
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.EventLog;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.FeeShareScore;
import foundation.icon.test.score.GovScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.fail;

public class FeeSharingTest extends TestBase {
    private static TransactionHandler txHandler;
    private static GovScore govScore;
    private static GovScore.Fee fee;
    private static KeyWallet ownerWallet;
    private static KeyWallet aliceWallet;
    private static ChainScore chainScore;

    @BeforeAll
    static void setup() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        ownerWallet = KeyWallet.create();
        aliceWallet = KeyWallet.create();
        LOG.info("Address of owner: " + ownerWallet.getAddress());
        LOG.info("Address of alice: " + aliceWallet.getAddress());

        // transfer initial icx to test addresses
        BigInteger ownerBalance = ICX.multiply(new BigInteger("5100")); // deploy(100) + deposit(5000)
        txHandler.transfer(chain.governorWallet.getAddress(), ownerBalance);
        txHandler.transfer(ownerWallet.getAddress(), ownerBalance);
        txHandler.transfer(aliceWallet.getAddress(), ICX);
        ensureIcxBalance(txHandler, ownerWallet.getAddress(), BigInteger.ZERO, ownerBalance);
        ensureIcxBalance(txHandler, aliceWallet.getAddress(), BigInteger.ZERO, ICX);

        LOG.infoEntering("initSteps");
        chainScore = new ChainScore(txHandler);
        govScore = new GovScore(txHandler);
        fee = govScore.getFee();
        StepTest.initStepCosts(govScore);
        LOG.infoExiting();
    }

    @AfterAll
    static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    private static BigInteger ensureIcxBalance(Address address, BigInteger val)
            throws IOException {
        BigInteger balance = txHandler.getBalance(address);
        LOG.info("ICX balance of " + address + ": " + balance);
        if (balance.compareTo(val) != 0) {
            throw new AssertionError("Balance mismatch!");
        }
        return balance;
    }

    @Tag(Constants.TAG_PY_GOV)
    @Test
    public void testPython() throws Exception {
        deployAndStartTest(Constants.CONTENT_TYPE_PYTHON);
    }

    @Tag(Constants.TAG_JAVA_GOV)
    @Test
    public void testJava() throws Exception {
        deployAndStartTest(Constants.CONTENT_TYPE_JAVA);
    }

    private void deployAndStartTest(String contentType) throws Exception {
        var feeScore = FeeShareScore.mustDeploy(txHandler, ownerWallet, contentType);
        for (int proportion : new int[]{0, 50, 100}) {
            LOG.infoEntering("Proportion: " + proportion);
            runTest(feeScore, BigInteger.valueOf(proportion));
            LOG.infoExiting();
        }
    }

    @Test
    public void runTest(FeeShareScore feeShareOwner, BigInteger proportion) throws Exception {
        // add alice into the white list
        LOG.infoEntering("invoke", "addToWhitelist(alice)");
        BigInteger ownerBalance = txHandler.getBalance(ownerWallet.getAddress());
        TransactionResult result = feeShareOwner.addToWhitelist(aliceWallet.getAddress(), proportion);
        assertSuccess(result);
        ownerBalance = subtractFee(ownerBalance, result);
        assertEquals(proportion, feeShareOwner.getProportion(aliceWallet.getAddress()));
        LOG.infoExiting();

        // set value before adding deposit (user balance should be decreased)
        LOG.infoEntering("invoke", "setValue() before adding deposit");
        FeeShareScore feeShareAlice = new FeeShareScore(feeShareOwner, aliceWallet);
        BigInteger aliceBalance = txHandler.getBalance(aliceWallet.getAddress());
        BigInteger treasuryBalance = txHandler.getBalance(Constants.TREASURY_ADDRESS);
        result = feeShareAlice.setValue("alice #1");
        assertSuccess(result);
        LOG.info("value: " + feeShareAlice.getValue());
        // check if the balance was decreased
        aliceBalance = ensureIcxBalance(aliceWallet.getAddress(), subtractFee(aliceBalance, result));
        // check the treasury balance
        treasuryBalance = ensureIcxBalance(Constants.TREASURY_ADDRESS, addFee(treasuryBalance, result));
        LOG.infoExiting();

        // add deposit 2000 to SCORE
        BigInteger depositAmount = IconAmount.of("2000", IconAmount.Unit.ICX).toLoop();
        LOG.infoEntering("addDeposit", depositAmount.toString());
        result = feeShareOwner.addDeposit(depositAmount);
        assertSuccess(result);
        printDepositInfo(feeShareOwner.getAddress());
        // check eventlog validity
        assertTrue(EventLog.checkScenario(List.of(
                new EventLog(feeShareOwner.getAddress().toString(),
                        "DepositAdded(bytes,Address,int,int)",
                        "0x", ownerWallet.getAddress().toString(),
                        "0x" + depositAmount.toString(16), "0x0")),
                result));
        // check the owner balance
        ownerBalance = subtractFee(ownerBalance.subtract(depositAmount), result);
        ensureIcxBalance(ownerWallet.getAddress(), ownerBalance);
        // check the SCORE balance
        ensureIcxBalance(feeShareOwner.getAddress(), BigInteger.ZERO);
        // check the treasury balance
        treasuryBalance = ensureIcxBalance(Constants.TREASURY_ADDRESS, addFee(treasuryBalance, result));
        LOG.infoExiting();

        // set value after adding deposit (user balance should be decreased by the proportion)
        LOG.infoEntering("invoke", "setValue() after adding deposit");
        result = feeShareAlice.setValue("alice #2");
        assertSuccess(result);
        printStepUsedDetails(result.getStepUsedDetails(), proportion);
        LOG.info("value: " + feeShareAlice.getValue());
        LOG.info("stepUsed: " + result.getStepUsed());

        // check if the balance was decreased by the proportion
        var stepUsedByScore = result.getStepUsed().multiply(proportion).divide(BigInteger.valueOf(100));
        var stepUsedByUser = result.getStepUsed().subtract(stepUsedByScore);
        LOG.info("stepUsed proportion: [" + stepUsedByScore + ", " + stepUsedByUser + "]");
        var depositRemain = depositAmount.subtract(stepUsedByScore.multiply(result.getStepPrice()));
        var userFee = stepUsedByUser.multiply(result.getStepPrice());
        aliceBalance = ensureIcxBalance(aliceWallet.getAddress(), aliceBalance.subtract(userFee));
        treasuryBalance = ensureIcxBalance(Constants.TREASURY_ADDRESS, addFee(treasuryBalance, result));
        printDepositInfo(feeShareOwner.getAddress());
        // check eventlog validity
        assertTrue(EventLog.checkScenario(List.of(
                new EventLog(feeShareOwner.getAddress().toString(),
                        "ValueSet(Address,int)",
                        aliceWallet.getAddress().toString(), "0x" + proportion.toString(16))),
                result));
        LOG.infoExiting();

        // withdraw the whole deposit
        LOG.infoEntering("withdrawDeposit", "amount=all");
        result = feeShareOwner.withdrawDeposit();
        assertSuccess(result);
        printDepositInfo(feeShareOwner.getAddress());
        // check eventlog validity
        assertTrue(EventLog.checkScenario(List.of(
                new EventLog(feeShareOwner.getAddress().toString(),
                        "DepositWithdrawn(bytes,Address,int,int)",
                        "0x", ownerWallet.getAddress().toString(),
                        "0x" + depositRemain.toString(16), "0x0")),
                result));
        // check the owner balance
        ownerBalance = subtractFee(ownerBalance.add(depositRemain), result);
        ensureIcxBalance(ownerWallet.getAddress(), ownerBalance);
        // check the SCORE balance
        ensureIcxBalance(feeShareOwner.getAddress(), BigInteger.ZERO);
        // check the treasury balance
        treasuryBalance = ensureIcxBalance(Constants.TREASURY_ADDRESS, addFee(treasuryBalance, result));
        LOG.infoExiting();

        // set value after withdrawing deposit (user balance should be decreased again)
        LOG.infoEntering("invoke", "setValue() after withdrawing deposit");
        result = feeShareAlice.setValue("alice #3");
        assertSuccess(result);
        assertNull(result.getStepUsedDetails());
        LOG.info("value: " + feeShareAlice.getValue());
        // check if the balance was decreased
        ensureIcxBalance(aliceWallet.getAddress(), subtractFee(aliceBalance, result));
        // check the treasury balance
        ensureIcxBalance(Constants.TREASURY_ADDRESS, addFee(treasuryBalance, result));
        LOG.infoExiting();
    }

    private BigInteger addFee(BigInteger balance, TransactionResult result) {
        BigInteger fee = result.getStepUsed().multiply(result.getStepPrice());
        return balance.add(fee);
    }

    private BigInteger subtractFee(BigInteger balance, TransactionResult result) {
        BigInteger fee = result.getStepUsed().multiply(result.getStepPrice());
        return balance.subtract(fee);
    }

    private void printStepUsedDetails(RpcItem stepUsedDetails, BigInteger proportion) {
        if (proportion.intValue() == 0) {
            assertNull(stepUsedDetails);
            return;
        }
        assertNotNull(stepUsedDetails);
        RpcObject details = stepUsedDetails.asObject();
        if (proportion.intValue() == 100) {
            assertEquals(1, details.keySet().size());
        } else {
            assertEquals(2, details.keySet().size());
        }
        LOG.info("stepUsedDetails: {");
        String M1 = "    ";
        for (String key : details.keySet()) {
            LOG.info(String.format(M1 + "%s: %s", key, details.getItem(key).asInteger()));
        }
        LOG.info("}");
    }

    private void printDepositInfo(Address scoreAddress) throws IOException {
        RpcItem status = chainScore.getScoreStatus(scoreAddress);
        RpcItem item = status.asObject().getItem("depositInfo");
        if (item != null) {
            LOG.info("depositInfo: {");
            RpcObject info = item.asObject();
            for (String key : info.keySet()) {
                String M1 = "    ";
                if (key.equals("deposits")) {
                    RpcArray deposits = info.getItem("deposits").asArray();
                    LOG.info(M1 + "deposits: {");
                    String M2 = M1 + M1;
                    RpcObject deposit = deposits.get(0).asObject();
                    for (String key2 : deposit.keySet()) {
                        if (key2.equals("id") || key2.equals("sender")) {
                            LOG.info(String.format(M2 + "%s: %s", key2, deposit.getItem(key2).asValue()));
                        } else {
                            LOG.info(String.format(M2 + "%s: %s", key2, deposit.getItem(key2).asInteger()));
                        }
                    }
                    LOG.info(M1 + "}");
                } else if (key.equals("scoreAddress")){
                    LOG.info(String.format(M1 + "%s: %s", key, info.getItem(key).asAddress()));
                } else {
                    LOG.info(String.format(M1 + "%s: %s", key, info.getItem(key).asInteger()));
                }
            }
            LOG.info("}");
        } else {
            LOG.info("depositInfo NULL");
        }
    }

    @Tag(Constants.TAG_JAVA_GOV)
    @Test
    public void testJavaSystemDeposit() throws Exception {
        testSystemDeposit(Constants.CONTENT_TYPE_JAVA);
    }

    @Tag(Constants.TAG_PY_GOV)
    @Test
    public void testPythonSystemDeposit() throws Exception {
        testSystemDeposit(Constants.CONTENT_TYPE_PYTHON);
    }

    @Test
    public void testSystemDeposit(String contentType) throws Exception {
        LOG.infoEntering("testSystemDeposit:" + contentType);
        var revision = chainScore.getRevision();
        if (revision < 8) {
            LOG.info("Nothing to test on revision=" + revision);
            LOG.infoExiting();
            return;
        }

        LOG.infoEntering("Ensure Revision 9");
        if (revision == 8) {
            LOG.info("setRevision(9)");
            var result = govScore.setRevision(9);
            assertSuccess(result);
        }
        LOG.infoExiting();

        LOG.infoEntering("deploy contracts");
        var feeScore1 = FeeShareScore.mustDeploy(txHandler, ownerWallet, contentType);
        var feeScore2 = FeeShareScore.mustDeploy(txHandler, ownerWallet, contentType);
        var feeScoreAlice = new FeeShareScore(feeScore1, aliceWallet);
        LOG.infoExiting();

        BigInteger aliceExpect = txHandler.getBalance(aliceWallet.getAddress());
        BigInteger aliceBalance;
        BigInteger systemDepositUsage = chainScore.getSystemDepositUsage();

        TransactionResult result;

        LOG.infoEntering("Case1: LAYER1(NO DEPOSIT) LAYER2(NO DEPOSIT)");

        LOG.info("Calling setValues(TEST) without system deposit");
        result = feeScoreAlice.setValues("TEST", new Address[]{feeScore2.getAddress()});
        assertSuccess(result);
        assertNull(result.getStepUsedDetails());
        aliceExpect = subtractFee(aliceExpect, result);
        aliceBalance = txHandler.getBalance(aliceWallet.getAddress());
        assertEquals(aliceExpect, aliceBalance);

        LOG.infoExiting();

        LOG.infoEntering("Case2: LAYER1(NO DEPOSIT) LAYER2(100% SYSTEM DEPOSIT)");

        LOG.info("Set UseSystemDeposit on " + feeScore2.getAddress().toString());
        // Fee of secondary contract is paid by the system
        govScore.setUseSystemDeposit(feeScore2.getAddress(), true);
        feeScore2.addToWhitelist(feeScore1.getAddress(), BigInteger.valueOf(100));

        LOG.info("Calling setValues()");
        result = feeScoreAlice.setValues("TEST", new Address[]{feeScore2.getAddress()});
        assertSuccess(result);

        LOG.info("Checking payment information");
        var feeMap = checkPayment(result, Map.of(
                Constants.CHAINSCORE_ADDRESS.toString(), 1,
                aliceWallet.getAddress().toString(), 1));
        systemDepositUsage = systemDepositUsage.add(feeMap.get(Constants.CHAINSCORE_ADDRESS.toString()));
        var sdu = chainScore.getSystemDepositUsage();
        assertEquals(systemDepositUsage, sdu);
        aliceExpect = aliceExpect.subtract(feeMap.get(aliceWallet.getAddress().toString()));
        aliceBalance = txHandler.getBalance(aliceWallet.getAddress());
        assertEquals(aliceExpect, aliceBalance);

        LOG.infoExiting();

        LOG.infoEntering("Case3: LAYER1(100% SYSTEM DEPOSIT) LAYER2(NO DEPOSIT)");

        LOG.info("Set UseSystemDeposit only on " + feeScore1.getAddress().toString());
        // All Fee is paid by system in the first contract
        govScore.setUseSystemDeposit(feeScore1.getAddress(), true);
        govScore.setUseSystemDeposit(feeScore2.getAddress(), false);
        feeScore1.addToWhitelist(aliceWallet.getAddress(), BigInteger.valueOf(100));

        LOG.info("Calling setValues()");
        result = feeScoreAlice.setValues("TEST", new Address[]{feeScore2.getAddress()});
        assertSuccess(result);

        LOG.info("Checking payment information");
        feeMap = checkPayment(result, Map.of(
                Constants.CHAINSCORE_ADDRESS.toString(), 1));
        systemDepositUsage = systemDepositUsage.add(feeMap.get(Constants.CHAINSCORE_ADDRESS.toString()));
        sdu = chainScore.getSystemDepositUsage();
        assertEquals(systemDepositUsage, sdu);

        LOG.infoExiting();

        LOG.infoEntering("Case4: NO -> SYSTEM -> SCORE -> NO");

        var feeScore3 = FeeShareScore.mustDeploy(txHandler, ownerWallet, contentType);
        BigInteger depositAmount = IconAmount.of("1000", IconAmount.Unit.ICX).toLoop();
        LOG.infoEntering("addDeposit", depositAmount.toString());
        result = feeScore3.addDeposit(depositAmount);
        assertSuccess(result);
        LOG.infoExiting();

        govScore.setUseSystemDeposit(feeScore1.getAddress(), false);
        govScore.setUseSystemDeposit(feeScore2.getAddress(), true);
        feeScore3.addToWhitelist(feeScore2.getAddress(), BigInteger.valueOf(100));
        feeScore1.addToWhitelist(feeScore3.getAddress(), BigInteger.valueOf(100));

        LOG.info("Calling setValues()");
        result = feeScoreAlice.setValues("TEST", new Address[]{
                feeScore2.getAddress(), feeScore3.getAddress(), feeScore1.getAddress()});
        assertSuccess(result);

        LOG.info("Checking payment information");
        feeMap = checkPayment(result, Map.of(
                Constants.CHAINSCORE_ADDRESS.toString(), 1,
                feeScore3.getAddress().toString(), 1,
                aliceWallet.getAddress().toString(), 1));
        systemDepositUsage = systemDepositUsage.add(feeMap.get(Constants.CHAINSCORE_ADDRESS.toString()));
        sdu = chainScore.getSystemDepositUsage();
        assertEquals(systemDepositUsage, sdu);
        aliceExpect = aliceExpect.subtract(feeMap.get(aliceWallet.getAddress().toString()));
        aliceBalance = txHandler.getBalance(aliceWallet.getAddress());
        assertEquals(aliceExpect, aliceBalance);
        LOG.infoExiting();

        LOG.infoEntering("withdrawDeposit");
        result = feeScore3.withdrawDeposit();
        assertSuccess(result);
        LOG.infoExiting();

        LOG.infoExiting();
    }

    private Map<String, BigInteger> checkPayment(TransactionResult result, Map<String, Integer> accMap) {
        var details = result.getStepUsedDetails().asObject();
        assertNotNull(details);
        Map<String, Integer> paymentCount = new HashMap<>();
        Map<String, BigInteger> feeMap = new HashMap<>();
        for (String addr : details.keySet()) {
            var steps = details.getItem(addr).asInteger();
            LOG.info("Payment payer=" + addr + " steps=" + steps.toString());
            var fee = details.getItem(addr).asInteger().multiply(result.getStepPrice());
            if (accMap.containsKey(addr)) {
                paymentCount.put(addr, paymentCount.getOrDefault(addr, 0) + 1);
                feeMap.put(addr, feeMap.getOrDefault(addr, BigInteger.ZERO).add(fee));
            } else {
                fail("Unknown Address:" + addr);
            }
        }
        assertEquals(paymentCount.size(), accMap.size());
        for (String addr : paymentCount.keySet()) {
            assertEquals(accMap.get(addr), paymentCount.get(addr));
        }
        return feeMap;
    }
}
