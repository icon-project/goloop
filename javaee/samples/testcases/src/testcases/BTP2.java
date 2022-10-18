/*
 * Copyright 2022 ICON Foundation
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
import score.annotation.External;

import java.math.BigInteger;

public class BTP2 {
    private Address bmc;

    public BTP2(Address address) {
        this.bmc = address;
    }

    @External
    public void sendAndRevert(BigInteger nid, byte[] msg, BigInteger msgCount, BigInteger revertNid) {
        try {
            Context.call(this.bmc, "sendMessageAndRevert", revertNid, msg);
        } catch (Exception e) {
            Context.println("[Exception] " + e.getMessage());
        }
        for (int i = 0; i < msgCount.intValue(); i++) {
            Context.call(this.bmc, "sendMessage", nid, msg);
        }
        try {
            Context.call(this.bmc, "sendMessageAndRevert", nid, msg);
        } catch (Exception e) {
            Context.println("[Exception] " + e.getMessage());
        }
    }
}
