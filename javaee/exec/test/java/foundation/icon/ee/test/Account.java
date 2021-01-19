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

package foundation.icon.ee.test;

import foundation.icon.ee.types.Address;
import org.aion.avm.core.util.ByteArrayWrapper;

import java.math.BigInteger;
import java.util.HashMap;
import java.util.Map;

public interface Account {
    Address getAddress();
    byte[] getStorage(byte[] key);
    byte[] setStorage(byte[] key, byte[] value);
    byte[] removeStorage(byte[] key);
    BigInteger getBalance();
    void setBalance(BigInteger balance);
    Contract getContract();
    byte[] getContractID();
}
