/*
 * Copyright (c) 2023 ICON Foundation
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
 *
 */

package testcases;

import score.Address;
import score.Context;
import score.annotation.EventLog;
import score.annotation.External;
import score.annotation.Payable;

import java.math.BigInteger;

public class EventGen {
    public EventGen(String name) {
        Context.println("on_install: name="+name);
    }

    @EventLog(indexed=3)
    public void Event(Address _addr, BigInteger _int, byte[] _bytes) {
    }

    @External
    public void generate(Address _addr, BigInteger _int, byte[] _bytes) {
        this.Event(_addr, _int, _bytes);
    }

    @EventLog(indexed=3)
    public void EventEx(boolean _bool, BigInteger _int, String _str, Address _addr, byte[] _bytes) {
    }

    @External
    public void generateNullByIndex(int _idx) {
        var args = new Object[]{
                true,
                BigInteger.valueOf(1),
                "test",
                Address.fromString("hx0000000000000000000000000000000000000000"),
                new byte[]{1}
        };
        args[_idx] = null;
        this.EventEx(
                (Boolean) args[0],
                (BigInteger) args[1],
                (String) args[2],
                (Address) args[3],
                (byte[]) args[4]
        );
    }

    @Payable
    public void fallback() {
        Context.println("fallback is called");
    }
}
