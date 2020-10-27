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

package example.token;

import score.Address;
import score.annotation.Optional;

import java.math.BigInteger;

public interface IRC2 {
    /**
     * Returns the name of the token. (e.g. "MySampleToken")
     */
    String name();

    /**
     * Returns the symbol of the token. (e.g. "MST")
     */
    String symbol();

    /**
     * Returns the number of decimals the token uses. (e.g. 18)
     */
    int decimals();

    /**
     * Returns the total token supply.
     */
    BigInteger totalSupply();

    /**
     * Returns the account balance of another account with address {@code _owner}.
     */
    BigInteger balanceOf(Address _owner);

    /**
     * Transfers {@code _value} amount of tokens to address {@code _to}, and MUST fire the {@code Transfer} event.
     * This function SHOULD throw if the caller account balance does not have enough tokens to spend.
     * If {@code _to} is a contract, this function MUST invoke the function {@code tokenFallback(Address, int, bytes)}
     * in {@code _to}. If the {@code tokenFallback} function is not implemented in {@code _to} (receiver contract),
     * then the transaction must fail and the transfer of tokens should not occur.
     * If {@code _to} is an externally owned address, then the transaction must be sent without trying to execute
     * {@code tokenFallback} in {@code _to}. {@code _data} can be attached to this token transaction.
     * {@code _data} can be empty.
     */
    void transfer(Address _to, BigInteger _value, @Optional byte[] _data);

    /**
     * (EventLog) Must trigger on any successful token transfers.
     */
    void Transfer(Address _from, Address _to, BigInteger _value, byte[] _data);
}
