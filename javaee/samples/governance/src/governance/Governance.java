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
import score.annotation.External;

import java.math.BigInteger;

public class Governance {
    private final SystemInterface system;

    public Governance(String name) {
        Context.println(">>> Governance: " + name);
        system = new SystemInterface();
    }

    @External
    public void setRevision(int code) {
        system.setRevision(code);
    }

    @External
    public void setStepPrice(BigInteger price) {
        system.setStepPrice(price);
    }

    @External
    public void setStepCost(String type, BigInteger cost) {
        system.setStepCost(type, cost);
    }

    @External
    public void setMaxStepLimit(String contextType, BigInteger limit) {
        system.setMaxStepLimit(contextType, limit);
    }

    @External
    public void acceptScore(byte[] txHash) {
        system.acceptScore(txHash);
    }

    @External
    public void rejectScore(byte[] txHash) {
        system.rejectScore(txHash);
    }

    @External
    public void blockScore(Address address) {
        system.blockScore(address);
    }

    @External
    public void unblockScore(Address address) {
        system.unblockScore(address);
    }

    @External
    public void addMember(Address address) {
        system.addMember(address);
    }

    @External
    public void removeMember(Address address) {
        system.removeMember(address);
    }

    @External
    public void grantValidator(Address address) {
        system.grantValidator(address);
    }

    @External
    public void revokeValidator(Address address) {
        system.revokeValidator(address);
    }

    @External
    public void addDeployer(Address address) {
        system.addDeployer(address);
    }

    @External
    public void removeDeployer(Address address) {
        system.removeDeployer(address);
    }

    @External
    public void setDeployerWhiteListEnabled(boolean yn) {
        system.setDeployerWhiteListEnabled(yn);
    }

    @External
    public void setTimestampThreshold(int threshold) {
        system.setTimestampThreshold(threshold);
    }

    @External
    public void setRoundLimitFactor(int factor) {
        system.setRoundLimitFactor(factor);
    }

    @External
    public void setUseSystemDeposit(Address address, boolean yn) {
        system.setUseSystemDeposit(address, yn);
    }

    @External
    public BigInteger openBTPNetwork(String networkTypeName, String name, Address owner) {
        return system.openBTPMessage(networkTypeName, name, owner);
    }

    @External
    public void closeBTPNetwork(BigInteger id) {
        system.closeBTPMessage(id);
    }
}
