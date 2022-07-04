/*
 * Copyright 2021 ICON Foundation
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

package governance;

import score.Address;
import score.Context;

import java.math.BigInteger;

class SystemInterface {
    private static final Address CHAIN_SCORE = Address.fromString("cx0000000000000000000000000000000000000000");

    private void validateHash(byte[] value) {
        Context.require(value.length == 32);
    }

    void setRevision(int code) {
        Context.call(CHAIN_SCORE, "setRevision", code);
    }

    void setStepPrice(BigInteger price) {
        Context.call(CHAIN_SCORE, "setStepPrice", price);
    }

    void setStepCost(String type, BigInteger cost) {
        Context.call(CHAIN_SCORE, "setStepCost", type, cost);
    }

    void setMaxStepLimit(String contextType, BigInteger limit) {
        Context.call(CHAIN_SCORE, "setMaxStepLimit", contextType, limit);
    }

    void acceptScore(byte[] txHash) {
        validateHash(txHash);
        Context.call(CHAIN_SCORE, "acceptScore", (Object) txHash);
    }

    void rejectScore(byte[] txHash) {
        validateHash(txHash);
        Context.call(CHAIN_SCORE, "rejectScore", (Object) txHash);
    }

    void blockScore(Address address) {
        Context.call(CHAIN_SCORE, "blockScore", address);
    }

    void unblockScore(Address address) {
        Context.call(CHAIN_SCORE, "unblockScore", address);
    }

    void addMember(Address address) {
        Context.call(CHAIN_SCORE, "addMember", address);
    }

    void removeMember(Address address) {
        Context.call(CHAIN_SCORE, "removeMember", address);
    }

    void grantValidator(Address address) {
        Context.call(CHAIN_SCORE, "grantValidator", address);
    }

    void revokeValidator(Address address) {
        Context.call(CHAIN_SCORE, "revokeValidator", address);
    }

    void addDeployer(Address address) {
        Context.call(CHAIN_SCORE, "addDeployer", address);
    }

    void removeDeployer(Address address) {
        Context.call(CHAIN_SCORE, "removeDeployer", address);
    }

    void setDeployerWhiteListEnabled(boolean yn) {
        Context.call(CHAIN_SCORE, "setDeployerWhiteListEnabled", yn);
    }

    void setTimestampThreshold(int threshold) {
        Context.call(CHAIN_SCORE, "setTimestampThreshold", threshold);
    }

    void setRoundLimitFactor(int factor) {
        Context.call(CHAIN_SCORE, "setRoundLimitFactor", factor);
    }

    void setUseSystemDeposit(Address address, boolean yn) {
        Context.call(CHAIN_SCORE, "setUseSystemDeposit", address, yn);
    }

    BigInteger openBTPMessage(String networkTypeName, String name, Address owner) {
        return (BigInteger) Context.call(CHAIN_SCORE, "openBTPNetwork", networkTypeName, name, owner);
    }

    void closeBTPMessage(BigInteger id) {
        Context.call(CHAIN_SCORE, "closeBTPNetwork", id);
    }
}
