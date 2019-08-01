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

package foundation.icon.score;

import org.aion.avm.core.IExternalState;
import org.aion.types.AionAddress;

import java.math.BigInteger;

public class ExternalState implements IExternalState {
    @Override
    public void commit() {

    }

    @Override
    public void commitTo(IExternalState externalState) {

    }

    @Override
    public IExternalState newChildExternalState() {
        return null;
    }

    @Override
    public void createAccount(AionAddress address) {

    }

    @Override
    public boolean hasAccountState(AionAddress address) {
        return false;
    }

    @Override
    public byte[] getCode(AionAddress address) {
        return new byte[0];
    }

    @Override
    public void putCode(AionAddress address, byte[] code) {

    }

    @Override
    public byte[] getTransformedCode(AionAddress address) {
        return new byte[0];
    }

    @Override
    public void setTransformedCode(AionAddress address, byte[] code) {

    }

    @Override
    public void putObjectGraph(AionAddress address, byte[] objectGraph) {

    }

    @Override
    public byte[] getObjectGraph(AionAddress address) {
        return new byte[0];
    }

    @Override
    public void putStorage(AionAddress address, byte[] key, byte[] value) {

    }

    @Override
    public void removeStorage(AionAddress address, byte[] key) {

    }

    @Override
    public byte[] getStorage(AionAddress address, byte[] key) {
        return new byte[0];
    }

    @Override
    public void deleteAccount(AionAddress address) {

    }

    @Override
    public BigInteger getBalance(AionAddress address) {
        return null;
    }

    @Override
    public void adjustBalance(AionAddress address, BigInteger amount) {

    }

    @Override
    public BigInteger getNonce(AionAddress address) {
        return null;
    }

    @Override
    public void incrementNonce(AionAddress address) {

    }

    @Override
    public void deductEnergyCost(AionAddress address, BigInteger cost) {

    }

    @Override
    public void refundAccount(AionAddress address, BigInteger refund) {

    }

    @Override
    public void payMiningFee(AionAddress address, BigInteger fee) {

    }

    @Override
    public byte[] getBlockHashByNumber(long blockNumber) {
        return new byte[0];
    }

    @Override
    public boolean accountNonceEquals(AionAddress address, BigInteger nonce) {
        return false;
    }

    @Override
    public boolean accountBalanceIsAtLeast(AionAddress address, BigInteger amount) {
        return false;
    }

    @Override
    public boolean isValidEnergyLimitForCreate(long limit) {
        return false;
    }

    @Override
    public boolean isValidEnergyLimitForNonCreate(long limit) {
        return false;
    }

    @Override
    public boolean destinationAddressIsSafeForThisVM(AionAddress address) {
        return false;
    }

    @Override
    public long getBlockNumber() {
        return 0;
    }

    @Override
    public long getBlockTimestamp() {
        return 0;
    }

    @Override
    public long getBlockEnergyLimit() {
        return 0;
    }

    @Override
    public long getBlockDifficulty() {
        return 0;
    }

    @Override
    public AionAddress getMinerAddress() {
        return null;
    }
}
