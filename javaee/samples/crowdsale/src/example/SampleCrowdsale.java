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

import score.Address;
import score.Context;
import score.DictDB;
import score.VarDB;
import score.annotation.EventLog;
import score.annotation.External;
import score.annotation.Payable;

import java.math.BigInteger;

public class SampleCrowdsale
{
    private static final BigInteger ONE_ICX = new BigInteger("1000000000000000000");
    private final Address beneficiary;
    private final Address tokenScore;
    private final BigInteger fundingGoal;
    private final long deadline;
    private boolean fundingGoalReached;
    private boolean crowdsaleClosed;
    private final DictDB<Address, BigInteger> balances;
    private final VarDB<BigInteger> amountRaised;

    public SampleCrowdsale(BigInteger _fundingGoalInIcx, Address _tokenScore, BigInteger _durationInBlocks) {
        // some basic requirements
        Context.require(_fundingGoalInIcx.compareTo(BigInteger.ZERO) >= 0);
        Context.require(_durationInBlocks.compareTo(BigInteger.ZERO) >= 0);

        this.beneficiary = Context.getCaller();
        this.fundingGoal = ONE_ICX.multiply(_fundingGoalInIcx);
        this.tokenScore = _tokenScore;
        this.deadline = Context.getBlockHeight() + _durationInBlocks.longValue();

        this.fundingGoalReached = false;
        this.crowdsaleClosed = true; // Crowdsale closed by default

        this.balances = Context.newDictDB("balances", BigInteger.class);
        this.amountRaised = Context.newVarDB("amountRaised", BigInteger.class);
    }

    /*
     * Receives initial tokens to reward to the contributors.
     */
    @External
    public void tokenFallback(Address _from, BigInteger _value, byte[] _data) {
        // check if the caller is a token SCORE address that this SCORE is interested in
        Context.require(Context.getCaller().equals(this.tokenScore));

        // depositing tokens can only be done by owner
        Context.require(Context.getOwner().equals(_from));

        // value should be greater than zero
        Context.require(_value.compareTo(BigInteger.ZERO) >= 0);

        // start Crowdsale hereafter
        Context.require(this.crowdsaleClosed);
        this.crowdsaleClosed = false;
        // emit eventlog
        CrowdsaleStarted(this.fundingGoal, this.deadline);
    }

    /*
     * Called when anyone sends funds to the SCORE and that funds would be regarded as a contribution.
     */
    @Payable
    public void fallback() {
        // check if the crowdsale is closed
        Context.require(!this.crowdsaleClosed);

        Address _from = Context.getCaller();
        BigInteger _value = Context.getValue();
        Context.require(_value.compareTo(BigInteger.ZERO) > 0);

        // accept the contribution
        BigInteger fromBalance = safeGetBalance(_from);
        this.balances.set(_from, fromBalance.add(_value));

        // increase the total amount of funding
        BigInteger amountRaised = safeGetAmountRaised();
        this.amountRaised.set(amountRaised.add(_value));

        // give tokens to the contributor as a reward
        byte[] _data = "called from Crowdsale".getBytes();
        Context.call(this.tokenScore, "transfer", _from, _value, _data);
        // emit eventlog
        FundTransfer(_from, _value, true);
    }

    /*
     * Checks if the goal has been reached and ends the campaign.
     */
    @External
    public void checkGoalReached() {
        if (afterDeadline()) {
            if (!this.crowdsaleClosed) {
                this.crowdsaleClosed = true;
                // emit eventlog
                CrowdsaleEnded();
            }
            BigInteger amountRaised = safeGetAmountRaised();
            if (amountRaised.compareTo(this.fundingGoal) >= 0) {
                this.fundingGoalReached = true;
                // emit eventlog
                GoalReached(this.beneficiary, amountRaised);
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
    public void safeWithdrawal() {
        if (afterDeadline()) {
            Address _from = Context.getCaller();

            // each contributor can withdraw the amount they contributed if the goal was not reached
            if (!this.fundingGoalReached) {
                BigInteger amount = safeGetBalance(_from);
                if (amount.compareTo(BigInteger.ZERO) > 0) {
                    // set their balance to ZERO first before transferring the amount to prevent reentrancy attack
                    this.balances.set(_from, BigInteger.ZERO);
                    // transfer the icx back to them
                    Context.transfer(_from, amount);
                    // emit eventlog
                    FundTransfer(_from, amount, false);
                }
            }

            // owner can withdraw the contribution since the sales target has been met.
            if (this.fundingGoalReached && this.beneficiary.equals(_from)) {
                BigInteger amountRaised = safeGetAmountRaised();
                if (amountRaised.compareTo(BigInteger.ZERO) > 0) {
                    // transfer the funds to beneficiary
                    Context.transfer(this.beneficiary, amountRaised);
                    // emit eventlog
                    FundTransfer(this.beneficiary, amountRaised, false);
                    // reset amountRaised
                    this.amountRaised.set(BigInteger.ZERO);
                }
            }
        }
    }

    private BigInteger safeGetBalance(Address owner) {
        return this.balances.getOrDefault(owner, BigInteger.ZERO);
    }

    private BigInteger safeGetAmountRaised() {
        return this.amountRaised.getOrDefault(BigInteger.ZERO);
    }

    private boolean afterDeadline() {
        // checks if it has been reached to the deadline block
        return Context.getBlockHeight() >= this.deadline;
    }

    @EventLog
    private void CrowdsaleStarted(BigInteger fundingGoal, long deadline) {}

    @EventLog
    private void CrowdsaleEnded() {}

    @EventLog(indexed=3)
    private void FundTransfer(Address backer, BigInteger amount, boolean isContribution) {}

    @EventLog(indexed=2)
    private void GoalReached(Address recipient, BigInteger totalAmountRaised) {}
}
