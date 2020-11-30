/*
 * Copyright 2018 ICON Foundation
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

package foundation.icon.test.score;

import example.SampleCrowdsale;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

public class CrowdSaleScore extends Score {
    public static CrowdSaleScore mustDeploy(TransactionHandler txHandler, Wallet owner,
                                            Address tokenAddress, BigInteger fundingGoalInIcx, String contentType)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        LOG.infoEntering("deploy", "Crowdsale");
        RpcObject params = new RpcObject.Builder()
                .put("_fundingGoalInIcx", new RpcValue(fundingGoalInIcx))
                .put("_tokenScore", new RpcValue(tokenAddress))
                .put("_durationInBlocks", new RpcValue(BigInteger.valueOf(10)))
                .build();
        Score score;
        if (contentType.equals(Constants.CONTENT_TYPE_PYTHON)) {
            score = txHandler.deploy(owner, getFilePath("sample_crowdsale"), params);
        } else if (contentType.equals(Constants.CONTENT_TYPE_JAVA)) {
            score = txHandler.deploy(owner, SampleCrowdsale.class, params);
        } else {
            throw new IllegalArgumentException("Unknown content type");
        }
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new CrowdSaleScore(score);
    }

    public CrowdSaleScore(Score other) {
        super(other);
    }

    public TransactionResult checkGoalReached(Wallet wallet)
            throws ResultTimeoutException, IOException {
        return invokeAndWaitResult(wallet, "checkGoalReached", null);
    }

    public TransactionResult safeWithdrawal(Wallet wallet)
            throws ResultTimeoutException, IOException {
        return invokeAndWaitResult(wallet, "safeWithdrawal", null);
    }

    public void ensureCheckGoalReached(Wallet wallet) throws Exception {
        while (true) {
            TransactionResult result = checkGoalReached(wallet);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new IOException("Failed to execute checkGoalReached.");
            }
            TransactionResult.EventLog event = findEventLog(result, "GoalReached(Address,int)");
            if (event != null) {
                break;
            }
            LOG.info("Sleep 1 second.");
            Thread.sleep(1000);
        }
    }

    public void ensureFundingGoal(TransactionResult result, BigInteger fundingGoalInIcx)
            throws IOException {
        TransactionResult.EventLog event = findEventLog(result, "CrowdsaleStarted(int,int)");
        if (event != null) {
            BigInteger fundingGoalInLoop = IconAmount.of(fundingGoalInIcx, IconAmount.Unit.ICX).toLoop();
            BigInteger fundingGoalFromScore = event.getData().get(0).asInteger();
            if (fundingGoalInLoop.equals(fundingGoalFromScore)) {
                return; // ensured
            }
        }
        throw new IOException("ensureFundingGoal failed.");
    }

    public void ensureFundTransfer(TransactionResult result, Address backer, BigInteger amount)
            throws IOException {
        TransactionResult.EventLog event = findEventLog(result, "FundTransfer(Address,int,bool)");
        if (event != null) {
            Address _backer = event.getIndexed().get(1).asAddress();
            BigInteger _amount = event.getIndexed().get(2).asInteger();
            Boolean isContribution = event.getIndexed().get(3).asBoolean();
            if (backer.equals(_backer) && amount.equals(_amount) && !isContribution) {
                return; // ensured
            }
        }
        throw new IOException("ensureFundTransfer failed.");
    }
}
