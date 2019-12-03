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

import avm.Address;
import avm.Blockchain;
import avm.DictDB;
import avm.Value;
import avm.ValueBuffer;
import foundation.icon.ee.tooling.abi.EventLog;
import foundation.icon.ee.tooling.abi.External;
import foundation.icon.ee.tooling.abi.Optional;

import java.math.BigInteger;

public class SampleToken
{
    private final String name;
    private final String symbol;
    private final int decimals;
    private final BigInteger totalSupply;
    private DictDB<Address> balances;

    private SampleToken(String name, String symbol, BigInteger decimals, BigInteger initialSupply) {
        this.name = name;
        this.symbol = symbol;
        this.decimals = decimals.intValue();

        // decimals must be larger than 0 and less than 21
        Blockchain.require(this.decimals >= 0);
        Blockchain.require(this.decimals <= 21);

        // initialSupply must be larger than 0
        Blockchain.require(initialSupply.compareTo(BigInteger.ZERO) >= 0);

        // calculate totalSupply
        if (initialSupply.compareTo(BigInteger.ZERO) > 0) {
            BigInteger oneToken = pow(BigInteger.TEN, this.decimals);
            this.totalSupply = oneToken.multiply(initialSupply);
        } else {
            this.totalSupply = BigInteger.ZERO;
        }

        // set the initial balance of the owner
        this.balances = Blockchain.newDictDB("balances");
        this.balances.set(Blockchain.getOrigin(), new ValueBuffer(this.totalSupply));
    }

    // BigInteger#pow() is not implemented in the shadow BigInteger.
    // we need to use our implementation for that.
    private static BigInteger pow(BigInteger base, int exponent) {
        BigInteger result = BigInteger.ONE;
        for (int i = 0; i < exponent; i++) {
            result = result.multiply(base);
        }
        return result;
    }

    private static SampleToken token;

    public static void onInstall(String _name,
                                 String _symbol,
                                 BigInteger _decimals,
                                 BigInteger _initialSupply) {
        token = new SampleToken(_name, _symbol, _decimals, _initialSupply);
    }

    @External(readonly=true)
    public static String name() {
        return token.name;
    }

    @External(readonly=true)
    public static String symbol() {
        return token.symbol;
    }

    @External(readonly=true)
    public static int decimals() {
        return token.decimals;
    }

    @External(readonly=true)
    public static BigInteger totalSupply() {
        return token.totalSupply;
    }

    @External(readonly=true)
    public static BigInteger balanceOf(Address _owner) {
        return safeGetBalance(_owner);
    }

    @External
    public static void transfer(Address _to, BigInteger _value, @Optional byte[] _data) {
        Address _from = Blockchain.getCaller();
        BigInteger fromBalance = safeGetBalance(_from);
        BigInteger toBalance = safeGetBalance(_to);

        // check some basic requirements
        Blockchain.require(_value.compareTo(BigInteger.ZERO) >= 0);
        Blockchain.require(fromBalance.compareTo(_value) >= 0);

        // adjust the balances
        safeSetBalance(_from, fromBalance.subtract(_value));
        safeSetBalance(_to, toBalance.add(_value));

        // if the recipient is SCORE, call 'tokenFallback' to handle further operation
        byte[] dataBytes = (_data == null) ? new byte[0] : _data;
        if (isContract(_to)) {
            Blockchain.call(_to, "tokenFallback", new Object[] {_from, _value, dataBytes}, BigInteger.ZERO);
        }

        // emit Transfer event
        Transfer(_from, _to, _value, dataBytes);
    }

    private static BigInteger safeGetBalance(Address owner) {
        Value v = token.balances.get(owner);
        return (v != null) ? v.asBigInteger() : BigInteger.ZERO;
    }

    private static void safeSetBalance(Address owner, BigInteger amount) {
        token.balances.set(owner, new ValueBuffer(amount));
    }

    private static boolean isContract(Address address) {
        byte[] ba = address.toByteArray();
        return (ba[0] == 0x1) ? true : false;
    }

    @EventLog(indexed=3)
    private static void Transfer(Address _from, Address _to, BigInteger _value, byte[] _data) {}
}
