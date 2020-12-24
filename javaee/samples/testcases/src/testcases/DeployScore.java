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

package testcases;

import score.Address;
import score.Context;
import score.annotation.EventLog;
import score.annotation.External;

public class DeployScore {
    private Address address;

    @External
    public void deploySingle(byte[] content) {
        try {
            this.address = Context.deploy(content);
            EmitScoreAddress(this.address);
        } catch (IllegalArgumentException e) {
            Context.revert(1, "Failed to deploy");
        }
    }

    @External
    public void deployMultiple(byte[] content) {
        var addr = Context.deploy(content);
        var addr1 = Context.deploy(content);
        Context.require(addr != addr1);
    }

    @External
    public void updateSingle(Address target, byte[] content, String name) {
        try {
            assert target != null;
            this.address = Context.deploy(target, content, name);
            Context.require(target.equals(this.address));
        } catch (IllegalArgumentException e) {
            Context.revert(2, "Failed to update");
        }
    }

    @External(readonly=true)
    public Address getOwner() {
        Context.require(this.address != null);
        return Context.call(Address.class, this.address, "getOwnerQuery");
    }

    @External(readonly=true)
    public Address getAddress() {
        Context.require(this.address != null);
        return Context.call(Address.class, this.address, "getAddressQuery");
    }

    @EventLog(indexed=1)
    private void EmitScoreAddress(Address address) {}
}
