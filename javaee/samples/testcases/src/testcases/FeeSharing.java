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
import score.DictDB;
import score.VarDB;
import score.annotation.EventLog;
import score.annotation.External;

public class FeeSharing {
    private final DictDB<Address, Integer> whitelist;
    private final VarDB<String> value;

    public FeeSharing() {
        whitelist = Context.newDictDB("whitelist", Integer.class);
        value = Context.newVarDB("value", String.class);
    }

    @External
    public void addToWhitelist(Address address, int proportion) {
        if (!Context.getCaller().equals(Context.getOwner())) {
            Context.revert("Not an owner");
        }
        whitelist.set(address, proportion);
    }

    @External(readonly=true)
    public int getProportion(Address address) {
        return whitelist.getOrDefault(address, 0);
    }

    @External(readonly=true)
    public String getValue() {
        return value.getOrDefault("No value");
    }

    @External
    public void setValue(String value) {
        this.value.set(value);
        int proportion = getProportion(Context.getCaller());
        Context.setFeeSharingProportion(proportion);
        ValueSet(Context.getCaller(), proportion);
    }

    @EventLog(indexed=1)
    public void ValueSet(Address address, int proportion) {}
}
