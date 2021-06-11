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
import foundation.icon.icx.Wallet;
import foundation.icon.icx.crypto.IconKeys;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.EventLog;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag(Constants.TAG_PY_SCORE)
@Tag(Constants.TAG_PY_GOV)
public class ScoreMethodTest {
    private static final String SCORE1_PATH = "method_caller";
    private static TransactionHandler txHandler;
    private static Wallet owner;
    private static Score methodCaller;

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        owner = chain.godWallet;

        methodCaller = txHandler.deploy(owner,
                Score.getFilePath(SCORE1_PATH), null);
    }

    @Test
    public void callInternalsDirectly() throws Exception {
        LOG.infoEntering("callInternalsDirectly");

        var testScore = methodCaller;

        LOG.infoEntering("send transactions");
        var txs = new ArrayList<Bytes>();
        txs.add(testScore.invoke(owner, "on_install", null));
        txs.add(testScore.invoke(owner, "on_update", null));
        txs.add(testScore.invoke(owner, "fallback", null));
        txs.add(testScore.invoke(owner, "fallback", null, BigInteger.valueOf(100)));
        LOG.infoExiting();

        LOG.infoEntering("check results");
        for (var tx : txs) {
            var result = txHandler.getResult(tx);
            assertEquals(result.getStatus(), Constants.STATUS_FAILURE);
        }
        LOG.infoExiting();

        LOG.infoExiting();
    }

    @Test
    public void checkInternalCalls() throws Exception {
        LOG.infoEntering("checkInternalCalls");

        LOG.infoEntering("on_install");
        var dtx = txHandler.deployOnly(owner, Score.getFilePath(SCORE1_PATH), null);
        var result = txHandler.getResult(dtx);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        // If the audit is enabled, it should be accepted to check
        // events.
        var score_addr = result.getScoreAddress();
        var acceptResult = txHandler.acceptScoreIfAuditEnabled(dtx);
        if (acceptResult != null) {
            result = acceptResult;
        }

        assertTrue(EventLog.checkScenario(List.of(
                new EventLog(score_addr, "Called(str,int)", "on_install")
        ), result));
        LOG.infoExiting();

        LOG.infoEntering("on_update");
        dtx = txHandler.deployOnly(owner, new Address(score_addr), Score.getFilePath(SCORE1_PATH), null);
        result = txHandler.getResult(dtx);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
        acceptResult = txHandler.acceptScoreIfAuditEnabled(dtx);
        if (acceptResult != null) {
            result = acceptResult;
        }
        assertTrue(EventLog.checkScenario(List.of(
                new EventLog(score_addr, "Called(str,int)", "on_update")
        ), result));
        LOG.infoExiting();

        LOG.infoEntering("fallback");
        Bytes tx = txHandler.transfer(new Address(score_addr), BigInteger.valueOf(1000));
        result = txHandler.getResult(tx);
        assertTrue(EventLog.checkScenario(List.of(
                new EventLog(score_addr, "Called(str,int)", "fallback")
        ), result));
        LOG.infoExiting();

        LOG.infoExiting();
    }

    @Test
    void callExternalCallFallback() throws Exception {
        LOG.infoEntering("callExternalCallFallback");
        var result = methodCaller.invokeAndWaitResult(owner, "callFallback", new RpcObject.Builder().put(
                "addr", new RpcValue(methodCaller.getAddress())
        ).build());
        assertTrue(EventLog.checkScenario(List.of(
            new EventLog(methodCaller.getAddress().toString(), "Called(str,int)", "fallback")
        ), result));
        LOG.infoExiting();
    }

    final static int RpcCodeBase = -30000;
    final static int ResultContractNotFound = 2;
    final static int ResultInvalidFormat = 5;
    final static int ResultAccessDenied = 9;
    final static int RpcInvalidFormat = RpcCodeBase - ResultInvalidFormat;
    final static int RpcAccessDenied = RpcCodeBase - ResultAccessDenied;

    private void checkQueryResult(String method, RpcObject params, int result) throws Exception {
        LOG.info("calling "+method);
        try {
            methodCaller.call(method, params);
            assertEquals(0, result);
        } catch (RpcError e) {
            var code = e.getCode();
            assertEquals(result, code);
        }
    }

    @Test void queryVariousMethods() throws Exception {
        var addr_params = new RpcObject.Builder().put(
                "addr", new RpcValue(methodCaller.getAddress())
        ).build();
        var owner_params = new RpcObject.Builder().put(
                "addr", new RpcValue(owner.getAddress())
        ).build();
        var int_params = new RpcObject.Builder().put(
                "_value", new RpcValue(BigInteger.ONE)
        ).build();

        checkQueryResult("externalDummy", null, RpcAccessDenied);
        checkQueryResult("payableDummy", null, RpcAccessDenied);
        checkQueryResult("externalWriteInt", int_params, RpcAccessDenied);
        checkQueryResult("externalEventLog", int_params, RpcAccessDenied);
        checkQueryResult("externalReturnInt", null, RpcAccessDenied);
        checkQueryResult("readonlyReturnInt", null, 0);
        checkQueryResult("readonlyWriteInt", int_params, RpcAccessDenied);
        checkQueryResult("readonlyEventLog", null, RpcInvalidFormat);
        checkQueryResult("readonlyTransfer", owner_params, RpcAccessDenied);
        checkQueryResult( "readonlyCallReadonlyReturnInt", addr_params, 0);
        checkQueryResult( "readonlyCallExternalDummy", addr_params, RpcAccessDenied);
    }

    @Test
    public void callReadonlyMethods() throws Exception {
        LOG.infoEntering("callReadonlyMethods");
        var addr_params = new RpcObject.Builder().put(
                "addr", new RpcValue(methodCaller.getAddress()))
                .build();
        var int_params = new RpcObject.Builder().put(
                "_value", new RpcValue(BigInteger.ONE))
                .build();

        var txs = new ArrayList<Bytes>();
        var expects = new ArrayList<BigInteger>();

        LOG.infoEntering("sendingTxs");
        LOG.info("case"+txs.size()+" readonlyReturnInt");
        txs.add(methodCaller.invoke(owner, "readonlyReturnInt", null));
        expects.add(BigInteger.ZERO);
        LOG.info("case"+txs.size()+" readonlyWriteInt");
        txs.add(methodCaller.invoke(owner, "readonlyWriteInt", int_params));
        expects.add(BigInteger.valueOf(ResultAccessDenied));
        LOG.info("case"+txs.size()+" readonlyEventLog");
        txs.add(methodCaller.invoke(owner, "readonlyEventLog", null));
        expects.add(BigInteger.valueOf(ResultInvalidFormat));
        LOG.info("case"+txs.size()+" readonlyCallReadonlyReturnInt");
        txs.add(methodCaller.invoke(owner, "readonlyCallReadonlyReturnInt", addr_params));
        expects.add(BigInteger.ZERO);
        LOG.info("case"+txs.size()+" readonlyCallExternalDummy");
        txs.add(methodCaller.invoke(owner, "readonlyCallExternalDummy", addr_params));
        expects.add(BigInteger.valueOf(ResultAccessDenied));
        LOG.info("case"+txs.size()+" externalCallReadonlyCallReadonlyReturnInt");
        txs.add(methodCaller.invoke(owner, "externalCallReadonlyCallReadonlyReturnInt", addr_params));
        expects.add(BigInteger.ZERO);
        LOG.info("case"+txs.size()+" externalCallReadonlyCallExternalDummy");
        txs.add(methodCaller.invoke(owner, "externalCallReadonlyCallExternalDummy", addr_params));
        expects.add(BigInteger.valueOf(ResultAccessDenied));
        LOG.info("case"+txs.size()+" externalCallReadonlyTransfer");
        txs.add(methodCaller.invoke(owner, "externalCallReadonlyTransfer", addr_params));
        expects.add(BigInteger.valueOf(ResultAccessDenied));
        LOG.infoExiting();

        LOG.infoEntering("checkResults");
        for (int i=0 ; i < txs.size() ; i++) {
            var tx = txs.get(i);
            var expect = expects.get(i);
            LOG.info("case"+i+" "+tx.toHexString(true));
            var result = txHandler.getResult(tx);
            if (expect.equals(BigInteger.ZERO)) {
                assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            } else {
                assertEquals(Constants.STATUS_FAILURE, result.getStatus());
                var failure = result.getFailure();
                assertEquals(expect, failure.getCode());
            }
        }
        LOG.infoExiting();

        LOG.infoExiting();
    }

    @Test
    void internalCallToEOA() throws Exception {
        LOG.infoEntering("internalCallToEOA");
        var tempWallet = KeyWallet.create();
        var invalidContract = new Address(Address.AddressPrefix.CONTRACT,
                IconKeys.getAddressHash(tempWallet.getPublicKey().toByteArray()));

        var params = new ArrayList<RpcObject>();
        var expects = new ArrayList<BigInteger>();
        // Success: valid contract
        params.add(new RpcObject.Builder()
                .put("addr", new RpcValue(methodCaller.getAddress()))
                .build());
        expects.add(BigInteger.ZERO);
        // Failure: invalid contract
        params.add(new RpcObject.Builder()
                .put("addr", new RpcValue(invalidContract))
                .build());
        expects.add(BigInteger.valueOf(ResultContractNotFound));
        // Success: existing EOA
        params.add(new RpcObject.Builder()
                .put("addr", new RpcValue(owner.getAddress()))
                .build());
        expects.add(BigInteger.ZERO);
        // Success: non-existing EOA
        params.add(new RpcObject.Builder()
                .put("addr", new RpcValue(tempWallet.getAddress()))
                .build());
        expects.add(BigInteger.ZERO);

        var txs = new ArrayList<Bytes>();
        for (RpcObject param : params) {
            txs.add(methodCaller.invoke(owner, "intercallProxy", param));
        }
        for (int i = 0 ; i < txs.size(); i++) {
            var tx = txs.get(i);
            var expect = expects.get(i);
            LOG.info("case" + i + ": " + tx.toHexString(true));
            var result = txHandler.getResult(tx);
            if (expect.equals(BigInteger.ZERO)) {
                assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            } else {
                assertEquals(Constants.STATUS_FAILURE, result.getStatus());
                var failure = result.getFailure();
                assertEquals(expect, failure.getCode());
            }
        }
        LOG.infoExiting();
    }
}
