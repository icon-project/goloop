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

package example;

import avm.Address;
import avm.Blockchain;
import avm.DictDB;
import avm.Value;
import avm.ValueBuffer;
import avm.VarDB;
import foundation.icon.ee.tooling.abi.EventLog;
import foundation.icon.ee.tooling.abi.External;
import foundation.icon.ee.tooling.abi.Payable;

import java.math.BigInteger;

public class SampleCrowdsale
{
    private static final BigInteger ONE_ICX = new BigInteger("1000000000000000000");
    private Address beneficiary;
    private Address tokenScore;
    private BigInteger fundingGoal;
    private long deadline;
    private boolean fundingGoalReached;
    private boolean crowdsaleClosed;
    private DictDB<Address> balances;
    private VarDB amountRaised;

    private SampleCrowdsale(BigInteger fundingGoalInIcx, Address tokenScore, BigInteger durationInBlocks) {
        this.beneficiary = Blockchain.getCaller();
        this.fundingGoal = ONE_ICX.multiply(fundingGoalInIcx);
        this.tokenScore = tokenScore;
        this.deadline = Blockchain.getBlockHeight() + durationInBlocks.longValue();

        this.fundingGoalReached = false;
        this.crowdsaleClosed = true; // Crowdsale closed by default

        this.balances = Blockchain.newDictDB("balances");
        this.amountRaised = Blockchain.newVarDB("amountRaised");
    }

    private static SampleCrowdsale crowdsale;

    public static void onInstall(BigInteger _fundingGoalInIcx,
                                 Address _tokenScore,
                                 BigInteger _durationInBlocks) {
        // some basic requirements
        Blockchain.require(_fundingGoalInIcx.compareTo(BigInteger.ZERO) >= 0);
        Blockchain.require(_durationInBlocks.compareTo(_durationInBlocks) >= 0);

        crowdsale = new SampleCrowdsale(_fundingGoalInIcx, _tokenScore, _durationInBlocks);
    }

    /*
     * Receives initial tokens to reward to the contributors.
     */
    @External
    public static void tokenFallback(Address _from, BigInteger _value, byte[] _data) {
        // check if the caller is a token SCORE address that this SCORE is interested in
        Blockchain.require(Blockchain.getCaller().equals(crowdsale.tokenScore));

        // depositing tokens can only be done by owner
        Blockchain.require(Blockchain.getOwner().equals(_from));

        // value should be greater than zero
        Blockchain.require(_value.compareTo(BigInteger.ZERO) >= 0);

        // start Crowdsale hereafter
        Blockchain.require(crowdsale.crowdsaleClosed);
        crowdsale.crowdsaleClosed = false;
        // emit eventlog
        CrowdsaleStarted(crowdsale.fundingGoal);
    }

    /*
     * Called when anyone sends funds to the SCORE and that funds would be regarded as a contribution.
     */
    @Payable
    public static void fallback() {
        // check if the crowdsale is closed
        Blockchain.require(crowdsale.crowdsaleClosed == false);

        Address _from = Blockchain.getCaller();
        BigInteger _value = Blockchain.getValue();
        Blockchain.require(_value.compareTo(BigInteger.ZERO) > 0);

        // accept the contribution
        BigInteger fromBalance = safeGetBalance(_from);
        crowdsale.balances.set(_from, new ValueBuffer(fromBalance.add(_value)));

        // increase the total amount of funding
        BigInteger amountRaised = safeGetAmountRaised();
        crowdsale.amountRaised.set(new ValueBuffer(amountRaised.add(_value)));

        // give tokens to the contributor as a reward
        byte[] _data = "called from Crowdsale".getBytes();
        Blockchain.call(crowdsale.tokenScore, "transfer", new Object[] {_from, _value, _data}, BigInteger.ZERO);
        // emit eventlog
        FundTransfer(_from, _value, true);
    }

    /*
     * Checks if the goal has been reached and ends the campaign.
     */
    @External
    public static void checkGoalReached() {
        if (afterDeadline()) {
            if (crowdsale.crowdsaleClosed == false) {
                crowdsale.crowdsaleClosed = true;
                // emit eventlog
                CrowdsaleEnded();
            }
            BigInteger amountRaised = safeGetAmountRaised();
            if (amountRaised.compareTo(crowdsale.fundingGoal) >= 0) {
                crowdsale.fundingGoalReached = true;
                // emit eventlog
                GoalReached(crowdsale.beneficiary, amountRaised);
            }
        }
    }

    /*
     * Withdraws the funds safely.
     *
     *  - If the funding goal has been reached, sends the entire amount to the beneficiary.
     *  - If the goal was not reached, each contributor can withdraw the amount they contributed.
     */
    @External
    public static void safeWithdrawal() {
        if (afterDeadline()) {
            Address _from = Blockchain.getCaller();

            // each contributor can withdraw the amount they contributed if the goal was not reached
            if (crowdsale.fundingGoalReached == false) {
                BigInteger amount = safeGetBalance(_from);
                if (amount.compareTo(BigInteger.ZERO) > 0) {
                    // transfer the icx back to them
                    Blockchain.call(_from, "fallback", new Object[0], amount);
                    // emit eventlog
                    FundTransfer(_from, amount, false);
                    // set their balance to ZERO
                    crowdsale.balances.set(_from, new ValueBuffer(BigInteger.ZERO));
                }
            }

            // owner can withdraw the contribution since the sales target has been met.
            if (crowdsale.fundingGoalReached && crowdsale.beneficiary.equals(_from)) {
                BigInteger amountRaised = safeGetAmountRaised();
                if (amountRaised.compareTo(BigInteger.ZERO) > 0) {
                    // transfer the funds to beneficiary
                    Blockchain.call(crowdsale.beneficiary, "fallback", new Object[0], amountRaised);
                    // emit eventlog
                    FundTransfer(crowdsale.beneficiary, amountRaised, false);
                    // reset amountRaised
                    crowdsale.amountRaised.set(new ValueBuffer(BigInteger.ZERO));
                }
            }
        }
    }

    private static BigInteger safeGetBalance(Address owner) {
        Value v = crowdsale.balances.get(owner);
        return (v != null) ? v.asBigInteger() : BigInteger.ZERO;
    }

    private static BigInteger safeGetAmountRaised() {
        Value v = crowdsale.amountRaised.get();
        return (v != null) ? v.asBigInteger() : BigInteger.ZERO;
    }

    private static boolean afterDeadline() {
        // checks if it has been reached to the deadline block
        return Blockchain.getBlockHeight() >= crowdsale.deadline;
    }

    @EventLog
    private static void CrowdsaleStarted(BigInteger fundingGoal) {}

    @EventLog
    private static void CrowdsaleEnded() {}

    @EventLog(indexed=3)
    private static void FundTransfer(Address backer, BigInteger amount, boolean isContribution) {}

    @EventLog(indexed=2)
    private static void GoalReached(Address recipient, BigInteger totalAmountRaised) {}
}
